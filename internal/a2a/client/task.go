package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/techbuzzz/agent-shaker/internal/a2a/models"
	"github.com/techbuzzz/agent-shaker/internal/task"
)

// SendMessage sends a task message to an external A2A agent
func (c *HTTPClient) SendMessage(ctx context.Context, agentURL string, req *models.SendMessageRequest) (*models.SendMessageResponse, error) {
	url := fmt.Sprintf("%s/a2a/v1/message", agentURL)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("send message failed: received status %d", resp.StatusCode)
	}

	var response models.SendMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

// GetTask retrieves a task from an external A2A agent
func (c *HTTPClient) GetTask(ctx context.Context, agentURL string, taskID string) (*models.Task, error) {
	url := fmt.Sprintf("%s/a2a/v1/tasks/%s", agentURL, taskID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("task %s not found", taskID)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get task failed: received status %d", resp.StatusCode)
	}

	var t models.Task
	if err := json.NewDecoder(resp.Body).Decode(&t); err != nil {
		return nil, fmt.Errorf("failed to decode task: %w", err)
	}

	return &t, nil
}

// ListTasks retrieves tasks from an external A2A agent
func (c *HTTPClient) ListTasks(ctx context.Context, agentURL string, filter *task.Filter) (*models.TaskListResponse, error) {
	url := fmt.Sprintf("%s/a2a/v1/tasks", agentURL)

	// Add query parameters
	if filter != nil {
		params := make([]string, 0)
		if filter.Status != "" {
			params = append(params, fmt.Sprintf("status=%s", filter.Status))
		}
		if filter.Limit > 0 {
			params = append(params, fmt.Sprintf("limit=%d", filter.Limit))
		}
		if filter.Offset > 0 {
			params = append(params, fmt.Sprintf("offset=%d", filter.Offset))
		}
		if len(params) > 0 {
			url += "?" + strings.Join(params, "&")
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list tasks failed: received status %d", resp.StatusCode)
	}

	var response models.TaskListResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

// StreamMessage sends a task and streams updates via SSE
func (c *HTTPClient) StreamMessage(ctx context.Context, agentURL string, req *models.SendMessageRequest) (<-chan task.TaskUpdate, error) {
	url := fmt.Sprintf("%s/a2a/v1/message:stream", agentURL)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("stream message failed: received status %d", resp.StatusCode)
	}

	updates := make(chan task.TaskUpdate, 10)

	go func() {
		defer resp.Body.Close()
		defer close(updates)

		scanner := bufio.NewScanner(resp.Body)
		var currentEvent string

		for scanner.Scan() {
			line := scanner.Text()

			// Handle event type
			if strings.HasPrefix(line, "event:") {
				currentEvent = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
				continue
			}

			// Handle data
			if strings.HasPrefix(line, "data:") {
				data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))

				var updateData map[string]any
				if err := json.Unmarshal([]byte(data), &updateData); err != nil {
					continue
				}

				update := task.TaskUpdate{
					Event: currentEvent,
					Data:  updateData,
				}

				// Check for final events
				if currentEvent == "completed" || currentEvent == "failed" || currentEvent == "cancelled" {
					update.IsFinal = true
				}

				select {
				case updates <- update:
				case <-ctx.Done():
					return
				}

				if update.IsFinal {
					return
				}
			}
		}
	}()

	return updates, nil
}

// ListArtifacts retrieves artifacts from an external A2A agent
func (c *HTTPClient) ListArtifacts(ctx context.Context, agentURL string) (*models.ArtifactListResponse, error) {
	url := fmt.Sprintf("%s/a2a/v1/artifacts", agentURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list artifacts failed: received status %d", resp.StatusCode)
	}

	var response models.ArtifactListResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

// GetArtifact retrieves a specific artifact from an external A2A agent
func (c *HTTPClient) GetArtifact(ctx context.Context, agentURL string, artifactID string) (*models.Artifact, error) {
	url := fmt.Sprintf("%s/a2a/v1/artifacts/%s", agentURL, artifactID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("artifact %s not found", artifactID)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get artifact failed: received status %d", resp.StatusCode)
	}

	var artifact models.Artifact
	if err := json.NewDecoder(resp.Body).Decode(&artifact); err != nil {
		return nil, fmt.Errorf("failed to decode artifact: %w", err)
	}

	return &artifact, nil
}
