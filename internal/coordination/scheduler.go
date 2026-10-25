package coordination

import (
	"context"
	"log/slog"
	"time"

	"github.com/go-co-op/gocron/v2"

	"github.com/techbuzzz/agent-shaker/internal/database/queries"
)

// RetentionStore is the slice of the heartbeats store the retention job needs.
type RetentionStore interface {
	DeleteOlderThan(ctx context.Context, before time.Time) (int64, error)
}

// Scheduler owns every recurring background job in the server.
//
// The reaper used to run its own time.Ticker inside Run(). Moving timing into
// one scheduler is what makes the job list inspectable — Jobs() says what runs
// and how often — and it is what lets the retention job exist without a second
// goroutine, a second ticker and a second shutdown path to remember.
//
// All jobs are bound to the caller's context, so one signal stops them all.
type Scheduler struct {
	scheduler gocron.Scheduler
	logger    *slog.Logger
}

// SchedulerConfig holds the timings of the background jobs.
type SchedulerConfig struct {
	// Reaper carries the lease TTL and the sweep interval.
	Reaper Config
	// HeartbeatRetention is how long heartbeat history is kept.
	HeartbeatRetention time.Duration
	// RetentionCron is when the retention job runs. Small hours keep it out of
	// the way of anything an operator might be watching.
	RetentionCron string
}

// DefaultSchedulerConfig returns the documented defaults: a 60s lease swept
// every 30s, and 30 days of heartbeat history pruned a little after 03:00.
func DefaultSchedulerConfig() SchedulerConfig {
	return SchedulerConfig{
		Reaper:             DefaultConfig(),
		HeartbeatRetention: 30 * 24 * time.Hour,
		RetentionCron:      "12 3 * * *",
	}
}

// NewScheduler registers the background jobs and returns the scheduler without
// starting it, so the caller controls when the work begins.
//
// A nil store (the server running without a database) or a nil reaper simply
// means that job is not registered: there is nothing to sweep and nothing to
// prune, and a scheduler that panics on startup would take the whole server
// with it.
func NewScheduler(ctx context.Context, reaper *StaleAgentReaper, heartbeats RetentionStore, cfg SchedulerConfig) (*Scheduler, error) {
	inner, err := gocron.NewScheduler()
	if err != nil {
		return nil, err
	}

	s := &Scheduler{scheduler: inner, logger: slog.Default().With("component", "scheduler")}

	if reaper != nil {
		interval := cfg.Reaper.ReaperInterval
		if interval <= 0 {
			interval = DefaultConfig().ReaperInterval
		}
		ttl := cfg.Reaper.LeaseTTL
		if ttl <= interval {
			// A lease at or below the sweep interval is not a faster feedback
			// loop; it is an agent declared dead between two heartbeats.
			ttl = interval * 2
			s.logger.Warn("lease TTL must exceed the sweep interval; clamping",
				"lease_ttl", ttl, "sweep_interval", interval)
		}
		if _, err := inner.NewJob(
			gocron.DurationJob(interval),
			gocron.NewTask(reaper.SweepOnce),
			gocron.WithContext(ctx),
			gocron.WithName("stale-agent-reaper"),
			gocron.WithSingletonMode(gocron.LimitModeReschedule),
		); err != nil {
			return nil, err
		}
	}

	if heartbeats != nil {
		retention := cfg.HeartbeatRetention
		if retention <= 0 {
			retention = DefaultSchedulerConfig().HeartbeatRetention
		}
		cron := cfg.RetentionCron
		if cron == "" {
			cron = DefaultSchedulerConfig().RetentionCron
		}
		if _, err := inner.NewJob(
			gocron.CronJob(cron, false),
			gocron.NewTask(func(jobCtx context.Context) { s.pruneHeartbeats(jobCtx, heartbeats, retention) }),
			gocron.WithContext(ctx),
			gocron.WithName("heartbeat-retention"),
			gocron.WithSingletonMode(gocron.LimitModeReschedule),
		); err != nil {
			return nil, err
		}
	}

	s.logger.Info("background jobs registered",
		"sweep_interval", cfg.Reaper.ReaperInterval,
		"lease_ttl", cfg.Reaper.LeaseTTL,
		"heartbeat_retention", cfg.HeartbeatRetention,
		"retention_cron", cfg.RetentionCron)
	return s, nil
}

// pruneHeartbeats deletes history past the retention window.
func (s *Scheduler) pruneHeartbeats(ctx context.Context, store RetentionStore, retention time.Duration) {
	cutoff := time.Now().Add(-retention)
	removed, err := store.DeleteOlderThan(ctx, cutoff)
	if err != nil {
		// A failed prune must not stop the scheduler: the next run tries again,
		// and the table growing is slower than the server going down.
		s.log().Error("heartbeat retention failed", "cutoff", cutoff, "error", err)
		return
	}
	if removed > 0 {
		s.log().Info("pruned heartbeat history", "rows", removed, "older_than", cutoff)
	}
}

// log returns the scheduler's logger, or a default one. A zero-value Scheduler
// is constructed in tests and is a reasonable thing to hold; it must not panic
// on the first job that has something to report.
func (s *Scheduler) log() *slog.Logger {
	if s.logger == nil {
		return slog.Default()
	}
	return s.logger
}

// Start begins running the registered jobs.
func (s *Scheduler) Start() { s.scheduler.Start() }

// Shutdown stops the scheduler and waits for a job in flight to finish. It is
// the graceful path: a reaper interrupted mid-transaction would leave an agent
// marked offline whose tasks were never released.
func (s *Scheduler) Shutdown() { _ = s.scheduler.Shutdown() }

// Jobs exposes the registered jobs, for logging and for tests.
func (s *Scheduler) Jobs() []gocron.Job { return s.scheduler.Jobs() }

var _ RetentionStore = (*queries.HeartbeatsStore)(nil)
