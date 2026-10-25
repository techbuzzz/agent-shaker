package hooks

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/techbuzzz/agent-shaker/internal/database/queries"
)

// StoreResolver reads the project context a hook template needs.
//
// It reads the project rather than carrying it on the event: the event knows
// which project changed, and the project row knows what the project is called.
//
// The working directory is configured rather than looked up. The
// project_repos table records a repository URL, not a path on disk, so there
// is no local checkout for the server to resolve — and guessing one would put
// a command in a directory nobody chose.
type StoreResolver struct {
	projects *queries.ProjectsStore
	workDir  string
}

// NewStoreResolver returns a resolver backed by the project store.
//
// The work directory is the directory hook commands run in; an empty value
// leaves them in the server's own working directory.
func NewStoreResolver(projects *queries.ProjectsStore, workDir string) *StoreResolver {
	return &StoreResolver{projects: projects, workDir: workDir}
}

// ResolveProject returns the project's name and the working directory its
// hooks run in.
func (r *StoreResolver) ResolveProject(ctx context.Context, projectID uuid.UUID) (ProjectContext, error) {
	project, err := r.projects.GetProject(ctx, projectID)
	if err != nil {
		return ProjectContext{}, fmt.Errorf("read project: %w", err)
	}

	return ProjectContext{
		ID:       project.ID.String(),
		Name:     project.Name,
		RepoPath: r.workDir,
	}, nil
}
