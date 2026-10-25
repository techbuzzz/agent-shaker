package hooks

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/techbuzzz/agent-shaker/internal/events"
	"github.com/techbuzzz/agent-shaker/internal/models"
)

// HookStore is the persistence the dispatcher needs.
type HookStore interface {
	ListEnabledHooks(ctx context.Context, projectID uuid.UUID, eventType string) ([]models.Hook, error)
	HooksEnabled(ctx context.Context, projectID uuid.UUID) (bool, error)
	RecordRun(ctx context.Context, run *models.HookRun) error
}

// ContextWriter writes a hook's output back into the domain as a context.
type ContextWriter interface {
	CreateContext(ctx context.Context, c *models.Context) error
}

// Dispatcher runs the hooks bound to a domain event.
//
// The publish path never blocks and never queries. An event is dropped into
// a queue; a resolver goroutine works out which hooks it applies to; the
// runner workers execute them. That shape is deliberate: the publish happens
// inside the request that caused the change, and a hook that took thirty
// seconds must not turn a task completion into a thirty-second request.
type Dispatcher struct {
	store    HookStore
	projects ProjectResolver
	contexts ContextWriter
	secrets  SecretResolver
	runner   *Runner
	cfg      Config
	clock    Clock

	queue chan events.Event
	jobs  chan hookJob

	mu         sync.Mutex
	perProject map[uuid.UUID]chan struct{}

	wg      sync.WaitGroup
	closing chan struct{}
	once    sync.Once
}

type hookJob struct {
	hook     models.Hook
	event    events.Event
	delivery uuid.UUID
}

// NewDispatcher starts a dispatcher and its workers.
//
// The dispatcher runs even when hooks are disabled: the workers simply find
// no work. Starting it unconditionally means the enabled/disabled decision
// lives in one place — HandleEvent — rather than in the wiring.
func NewDispatcher(
	cfg Config,
	store HookStore,
	projects ProjectResolver,
	contexts ContextWriter,
	secrets SecretResolver,
) *Dispatcher {
	d := &Dispatcher{
		store:      store,
		projects:   projects,
		contexts:   contexts,
		secrets:    secrets,
		runner:     NewRunner(cfg),
		cfg:        cfg,
		clock:      SystemClock{},
		queue:      make(chan events.Event, cfg.QueueSize),
		jobs:       make(chan hookJob, cfg.QueueSize),
		perProject: make(map[uuid.UUID]chan struct{}),
		closing:    make(chan struct{}),
	}

	d.wg.Add(1)
	go d.resolveLoop()

	for i := 0; i < cfg.MaxConcurrency; i++ {
		d.wg.Add(1)
		go d.runLoop()
	}

	return d
}

// HandleEvent queues an event for the hooks bound to it.
//
// It never blocks and never fails: the caller is a request handler, and a
// full queue means the deployment is already running more hooks than it can
// keep up with, which is a condition to report rather than to propagate into
// the request.
func (d *Dispatcher) HandleEvent(ctx context.Context, ev events.Event) {
	if d == nil || !d.cfg.Enabled {
		return
	}
	if !hookableEvent(ev.Type) {
		return
	}

	select {
	case d.queue <- ev:
	case <-d.closing:
	default:
		slog.WarnContext(ctx, "hook queue is full, dropping event",
			"event_type", ev.Type,
			"project_id", ev.ProjectID,
		)
	}
}

// hookableEvent reports whether any hook may bind to this event type.
func hookableEvent(eventType string) bool {
	for _, known := range events.KnownTypes() {
		if known == eventType {
			return true
		}
	}
	return false
}

// resolveLoop turns events into jobs.
func (d *Dispatcher) resolveLoop() {
	defer d.wg.Done()

	for {
		select {
		case <-d.closing:
			return
		case ev := <-d.queue:
			d.resolve(ev)
		}
	}
}

// resolve finds the hooks an event applies to and queues them.
//
// The project lookup and the hook lookup happen here rather than in the
// publish path, for the same reason: they are database round trips, and the
// request that caused the event should not pay for them.
func (d *Dispatcher) resolve(ev events.Event) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	enabled, err := d.store.HooksEnabled(ctx, ev.ProjectID)
	if err != nil {
		slog.ErrorContext(ctx, "hook dispatcher could not read the project opt-in",
			"error", err,
			"project_id", ev.ProjectID,
		)
		return
	}
	if !enabled {
		return
	}

	hooks, err := d.store.ListEnabledHooks(ctx, ev.ProjectID, ev.Type)
	if err != nil {
		slog.ErrorContext(ctx, "hook dispatcher could not list hooks",
			"error", err,
			"event_type", ev.Type,
			"project_id", ev.ProjectID,
		)
		return
	}

	delivery := uuid.New()
	for _, hook := range hooks {
		matcher, err := CompileMatcher(hook.Matcher)
		if err != nil {
			slog.ErrorContext(ctx, "hook matcher does not compile, skipping hook",
				"error", err,
				"hook_id", hook.ID,
			)
			continue
		}
		if !matcher.Matches(ev.Payload) {
			continue
		}

		job := hookJob{hook: hook, event: ev, delivery: delivery}
		select {
		case d.jobs <- job:
		case <-d.closing:
			return
		}
	}
}

// runLoop executes queued jobs.
func (d *Dispatcher) runLoop() {
	defer d.wg.Done()

	for {
		select {
		case <-d.closing:
			return
		case job := <-d.jobs:
			d.execute(job)
		}
	}
}

// execute runs one hook and records what happened.
func (d *Dispatcher) execute(job hookJob) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	release, ok := d.acquireProject(job.event.ProjectID)
	if !ok {
		slog.WarnContext(ctx, "hook skipped: project concurrency limit reached",
			"hook_id", job.hook.ID,
			"project_id", job.event.ProjectID,
		)
		return
	}
	defer release()

	result, run := d.runHook(ctx, job)
	if err := d.store.RecordRun(ctx, &run); err != nil {
		slog.ErrorContext(ctx, "hook run could not be recorded",
			"error", err,
			"hook_id", job.hook.ID,
		)
	}

	if result.Status != models.HookRunSucceeded {
		return
	}
	d.post(ctx, job, result)
}

// runHook prepares and executes one hook, and returns both the outcome and
// the run row to record.
func (d *Dispatcher) runHook(ctx context.Context, job hookJob) (RunResult, models.HookRun) {
	run := models.HookRun{
		ID:         uuid.New(),
		HookID:     &job.hook.ID,
		DeliveryID: job.delivery,
		ProjectID:  job.event.ProjectID,
		Event:      job.event.Type,
		EntityType: job.event.EntityType,
		EntityID:   entityID(job.event),
		StartedAt:  d.clock.Now(),
	}

	renderCtx, err := d.renderContext(ctx, job)
	if err != nil {
		run.Status = models.HookRunFailed
		run.Stderr = err.Error()
		finished := d.clock.Now()
		run.FinishedAt = &finished
		return RunResult{Status: run.Status}, run
	}

	argv, err := RenderArgv(job.hook.Command, renderCtx)
	if err != nil {
		run.Status = models.HookRunFailed
		run.Stderr = err.Error()
		finished := d.clock.Now()
		run.FinishedAt = &finished
		return RunResult{Status: run.Status}, run
	}

	stdin, err := d.buildStdin(job.hook, renderCtx, job.event)
	if err != nil {
		run.Status = models.HookRunFailed
		run.Stderr = err.Error()
		finished := d.clock.Now()
		run.FinishedAt = &finished
		return RunResult{Status: run.Status}, run
	}

	env, secrets := d.buildEnv(job.hook, renderCtx)

	result, runErr := d.runner.Run(ctx, RunRequest{
		Argv:    argv,
		Stdin:   stdin,
		WorkDir: renderCtx.Project.RepoPath,
		Env:     env,
		Timeout: time.Duration(job.hook.TimeoutSeconds) * time.Second,
		Secrets: secrets,
	})
	if runErr != nil {
		run.Status = models.HookRunFailed
		run.Stderr = runErr.Error()
		finished := d.clock.Now()
		run.FinishedAt = &finished
		return RunResult{Status: run.Status}, run
	}

	exitCode := result.ExitCode
	duration := result.DurationMS
	finished := d.clock.Now()

	run.Status = Classify(result.ExitCode, result.TimedOut, false)
	run.ExitCode = &exitCode
	run.Stdout = result.Stdout
	run.Stderr = result.Stderr
	run.DurationMS = &duration
	run.FinishedAt = &finished

	result.Status = run.Status
	return result, run
}

// buildStdin produces the command's standard input.
//
// The two modes exist because the two kinds of consumer want different
// things. A script wants the event, as JSON, and can read whatever it needs
// out of it. A headless agent CLI wants a prompt — it has no idea what a
// "task.completed event" is, and it will not read one.
func (d *Dispatcher) buildStdin(hook models.Hook, rc RenderContext, ev events.Event) ([]byte, error) {
	switch hook.Stdin.Mode {
	case models.HookStdinPrompt:
		prompt, err := Render(hook.Stdin.Template, rc)
		if err != nil {
			return nil, fmt.Errorf("render hook prompt: %w", err)
		}
		return []byte(prompt), nil
	case models.HookStdinEvent, "":
		return RenderEventJSON(eventPayload(ev))
	default:
		return nil, fmt.Errorf("unknown hook stdin mode %q", hook.Stdin.Mode)
	}
}

// eventPayload is the JSON document a hook in "event" mode receives.
func eventPayload(ev events.Event) map[string]any {
	payload := map[string]any{
		"event":       ev.Type,
		"project_id":  ev.ProjectID.String(),
		"occurred_at": ev.OccurredAt.Format(time.RFC3339),
	}
	if ev.EntityType != "" {
		payload["entity_type"] = ev.EntityType
	}
	if ev.EntityID != uuid.Nil {
		payload["entity_id"] = ev.EntityID.String()
	}
	if ev.ActorType != "" {
		payload["actor_type"] = ev.ActorType
	}
	if ev.ActorID != uuid.Nil {
		payload["actor_id"] = ev.ActorID.String()
	}
	for k, v := range ev.Payload {
		payload[k] = v
	}
	return payload
}

// buildEnv builds the command's environment and returns the secret values in
// it, so the runner can redact them from the captured output.
func (d *Dispatcher) buildEnv(hook models.Hook, rc RenderContext) ([]string, []string) {
	env := baseEnv()
	secrets := make([]string, 0, len(hook.Env))

	env = append(env,
		"AGENT_SHAKER_EVENT="+rc.Event,
		"AGENT_SHAKER_PROJECT_ID="+rc.Project.ID,
		"AGENT_SHAKER_PROJECT_NAME="+rc.Project.Name,
		"AGENT_SHAKER_PROJECT_REPO_PATH="+rc.Project.RepoPath,
		"AGENT_SHAKER_ENTITY_ID="+rc.Entity.ID,
		"AGENT_SHAKER_ENTITY_TYPE="+rc.Entity.Type,
		"AGENT_SHAKER_ENTITY_TITLE="+rc.Entity.Title,
		"AGENT_SHAKER_ENTITY_STATUS="+rc.Entity.Status,
		"AGENT_SHAKER_ACTOR_ID="+rc.Actor.ID,
		"AGENT_SHAKER_ACTOR_NAME="+rc.Actor.Name,
		"AGENT_SHAKER_ACTOR_ROLE="+rc.Actor.Role,
		"AGENT_SHAKER_DELIVERY_ID="+rc.DeliveryID,
	)

	for name, value := range hook.Env {
		if hook.Env.IsSecretReference(name) {
			secretName, _ := hook.Env.SecretName(name)
			resolved, ok := d.resolveSecret(name, secretName)
			if !ok {
				continue
			}
			env = append(env, name+"="+resolved)
			secrets = append(secrets, resolved)
			continue
		}
		env = append(env, name+"="+value)
	}

	return env, secrets
}

func (d *Dispatcher) resolveSecret(varName, secretName string) (string, bool) {
	if d.secrets == nil {
		return "", false
	}
	return d.secrets.Resolve(context.Background(), secretName)
}

// baseEnv is the environment every hook command starts with.
//
// It is deliberately not the server's own environment. A hook that inherited
// it would inherit the server's database credentials, its OTLP endpoint and
// every secret the deployment holds.
//
// PATH is included because a hook's executable is usually a name rather than
// an absolute path, and the allow-list is what decides which name may run.
func baseEnv() []string {
	keys := []string{
		"PATH", "HOME", "USERPROFILE", "TMPDIR", "TEMP", "TMP",
		"SystemRoot", "SystemDrive", "ComSpec", "PATHEXT",
		"LOCALAPPDATA", "APPDATA",
	}

	env := make([]string, 0, len(keys))
	for _, key := range keys {
		if value, ok := os.LookupEnv(key); ok {
			env = append(env, key+"="+value)
		}
	}
	return env
}

// renderContext builds the namespace a hook's templates resolve against.
func (d *Dispatcher) renderContext(ctx context.Context, job hookJob) (RenderContext, error) {
	project, err := d.projects.ResolveProject(ctx, job.event.ProjectID)
	if err != nil {
		return RenderContext{}, fmt.Errorf("resolve project: %w", err)
	}

	payload := job.event.Payload
	rc := RenderContext{
		Event: job.event.Type,
		Project: ProjectContext{
			ID:       project.ID,
			Name:     project.Name,
			RepoPath: project.RepoPath,
		},
		Entity: EntityContext{
			ID:     uuidString(job.event.EntityID),
			Type:   job.event.EntityType,
			Title:  payloadString(payload, events.PayloadTitle),
			Status: payloadString(payload, events.PayloadStatus),
			Output: payloadString(payload, events.PayloadOutput),
		},
		Actor: ActorContext{
			ID:   uuidString(job.event.ActorID),
			Name: payloadString(payload, events.PayloadActorName),
			Role: payloadString(payload, events.PayloadActorRole),
		},
		DeliveryID: job.delivery.String(),
		OccurredAt: job.event.OccurredAt.Format(time.RFC3339),
	}

	return rc, nil
}

// post writes the hook's output back into the domain as a context.
//
// This is what turns a hook from a notification into a participant: the
// verdict lands on the task, where the agent that did the work reads it on
// its next turn.
func (d *Dispatcher) post(ctx context.Context, job hookJob, result RunResult) {
	if job.hook.Post == nil || job.hook.Post.Kind != models.HookPostKindContext {
		return
	}
	if d.contexts == nil {
		return
	}
	if job.event.ActorType != events.ActorAgent || job.event.ActorID == uuid.Nil {
		slog.InfoContext(ctx, "hook output not posted: the event has no agent to own the context")
		return
	}

	title, err := Render(job.hook.Post.Title, RenderContext{})
	if err != nil {
		title = job.hook.Post.Title
	}

	now := d.clock.Now()
	context := &models.Context{
		ID:        uuid.New(),
		ProjectID: job.event.ProjectID,
		AgentID:   job.event.ActorID,
		Title:     title,
		Content:   result.Stdout,
		Tags:      []string{},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := d.contexts.CreateContext(ctx, context); err != nil {
		slog.ErrorContext(ctx, "hook output could not be posted as a context",
			"error", err,
			"hook_id", job.hook.ID,
		)
	}
}

// acquireProject takes a slot in the project's concurrency limit.
func (d *Dispatcher) acquireProject(projectID uuid.UUID) (func(), bool) {
	d.mu.Lock()
	sem, ok := d.perProject[projectID]
	if !ok {
		sem = make(chan struct{}, d.cfg.PerProjectConcurrency)
		d.perProject[projectID] = sem
	}
	d.mu.Unlock()

	select {
	case sem <- struct{}{}:
		return func() { <-sem }, true
	default:
		return nil, false
	}
}

// Close stops the workers and waits for them to finish.
func (d *Dispatcher) Close() {
	d.once.Do(func() {
		close(d.closing)
	})
	d.wg.Wait()
}

// TestHook runs a hook against a synthetic payload, without touching real
// work.
//
// It is the endpoint that makes hooks usable: a hook that can only be
// validated by firing it at real work is a hook nobody will enable.
func (d *Dispatcher) TestHook(ctx context.Context, hook models.Hook, ev events.Event) (models.HookRun, error) {
	job := hookJob{hook: hook, event: ev, delivery: uuid.New()}

	_, run := d.runHook(ctx, job)
	if err := d.store.RecordRun(ctx, &run); err != nil {
		return run, fmt.Errorf("record hook test run: %w", err)
	}
	return run, nil
}

// entityID returns the entity id, or nil when the event is not about a row.
func entityID(ev events.Event) *uuid.UUID {
	if ev.EntityID == uuid.Nil {
		return nil
	}
	id := ev.EntityID
	return &id
}

func payloadString(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	value, ok := payload[key]
	if !ok || value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	return fmt.Sprintf("%v", value)
}

func uuidString(id uuid.UUID) string {
	if id == uuid.Nil {
		return ""
	}
	return id.String()
}
