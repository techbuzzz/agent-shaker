package client

import (
	"context"
	"net/http"
	"time"

	"github.com/techbuzzz/agent-shaker/internal/a2a/models"
	"github.com/techbuzzz/agent-shaker/internal/task"
)

// Client defines the interface for A2A client operations
type Client interface {
	Discover(ctx context.Context, agentURL string) (*models.AgentCard, error)
	SendMessage(ctx context.Context, agentURL string, req *models.SendMessageRequest) (*models.SendMessageResponse, error)
	GetTask(ctx context.Context, agentURL string, taskID string) (*models.Task, error)
	ListTasks(ctx context.Context, agentURL string, filter *task.Filter) (*models.TaskListResponse, error)
	StreamMessage(ctx context.Context, agentURL string, req *models.SendMessageRequest) (<-chan task.TaskUpdate, error)
	ListArtifacts(ctx context.Context, agentURL string) (*models.ArtifactListResponse, error)
	GetArtifact(ctx context.Context, agentURL string, artifactID string) (*models.Artifact, error)
}

// HTTPClient implements Client using HTTP
type HTTPClient struct {
	httpClient *http.Client
	userAgent  string
}

// ClientOption defines a function for configuring the HTTP client
type ClientOption func(*HTTPClient)

// WithTimeout sets the HTTP client timeout
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *HTTPClient) {
		c.httpClient.Timeout = timeout
	}
}

// WithUserAgent sets the User-Agent header
func WithUserAgent(userAgent string) ClientOption {
	return func(c *HTTPClient) {
		c.userAgent = userAgent
	}
}

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *HTTPClient) {
		c.httpClient = httpClient
	}
}

// NewHTTPClient creates a new A2A HTTP client
func NewHTTPClient(opts ...ClientOption) *HTTPClient {
	client := &HTTPClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		userAgent: "AgentShaker-A2A-Client/1.0",
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}
