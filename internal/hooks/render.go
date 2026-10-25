package hooks

import (
	"encoding/json"
	"fmt"
	"strings"
)

// RenderContext is the namespace a hook template is resolved against.
//
// It is deliberately a fixed set of names rather than the raw event. A
// template that could reach into anything the server happens to carry would
// couple every hook to the shape of every event, and a hook written against
// one server version would break on the next.
type RenderContext struct {
	Event      string
	Project    ProjectContext
	Entity     EntityContext
	Actor      ActorContext
	DeliveryID string
	OccurredAt string
}

// EntityContext is the entity an event is about.
type EntityContext struct {
	ID     string
	Type   string
	Title  string
	Status string
	Output string
}

// ActorContext is who or what caused the event.
type ActorContext struct {
	ID   string
	Name string
	Role string
}

// resolve looks up a dotted path in the context.
//
// The set is closed. An unknown path is an error rather than an empty
// string, because a template that silently renders nothing is a template that
// looks like it worked.
func (rc RenderContext) resolve(path string) (string, bool) {
	switch path {
	case "event":
		return rc.Event, true
	case "delivery_id":
		return rc.DeliveryID, true
	case "occurred_at":
		return rc.OccurredAt, true
	case "project.id":
		return rc.Project.ID, true
	case "project.name":
		return rc.Project.Name, true
	case "project.repo_path":
		return rc.Project.RepoPath, true
	case "entity.id":
		return rc.Entity.ID, true
	case "entity.type":
		return rc.Entity.Type, true
	case "entity.title":
		return rc.Entity.Title, true
	case "entity.status":
		return rc.Entity.Status, true
	case "entity.output":
		return rc.Entity.Output, true
	case "actor.id":
		return rc.Actor.ID, true
	case "actor.name":
		return rc.Actor.Name, true
	case "actor.role":
		return rc.Actor.Role, true
	default:
		return "", false
	}
}

// Render resolves the {{...}} placeholders in text.
//
// The whole text is returned unchanged when it carries no placeholder, so a
// literal string is never touched.
func Render(text string, rc RenderContext) (string, error) {
	if !strings.Contains(text, "{{") {
		return text, nil
	}

	var out strings.Builder
	rest := text

	for {
		start := strings.Index(rest, "{{")
		if start < 0 {
			out.WriteString(rest)
			break
		}
		out.WriteString(rest[:start])

		rest = rest[start+2:]
		end := strings.Index(rest, "}}")
		if end < 0 {
			return "", fmt.Errorf("unterminated placeholder in %q", text)
		}

		path := strings.TrimSpace(rest[:end])
		value, ok := rc.resolve(path)
		if !ok {
			return "", fmt.Errorf("unknown placeholder %q", path)
		}
		out.WriteString(value)
		rest = rest[end+2:]
	}

	return out.String(), nil
}

// RenderArgv resolves the placeholders in every element of an argument
// vector.
//
// Each element is rendered on its own, so a placeholder that resolves to a
// path containing a space stays one argument. Rendering the whole vector as
// one string and splitting it afterwards would be the wrong way round: it
// would re-introduce the quoting problem the argument vector exists to
// avoid.
func RenderArgv(argv []string, rc RenderContext) ([]string, error) {
	out := make([]string, 0, len(argv))

	for i, arg := range argv {
		rendered, err := Render(arg, rc)
		if err != nil {
			return nil, fmt.Errorf("argument %d: %w", i, err)
		}
		out = append(out, rendered)
	}

	return out, nil
}

// RenderEventJSON encodes the whole event for the "event" stdin mode.
func RenderEventJSON(payload map[string]any) ([]byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode hook event: %w", err)
	}
	return raw, nil
}
