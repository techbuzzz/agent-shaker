package httpx

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// A caller that posts a body the validator rejects made a mistake, not the
// server. This answers 500 "internal" for every invalid field in the API,
// which tells the caller to retry and hides the reason they should not.
func TestWriteValidationErrorAnswers400WithTheReason(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/tasks", nil)

	WriteValidationError(rec, req, fmt.Errorf("agent_id is required"))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{`"code":"bad_request"`, "agent_id is required"} {
		if !strings.Contains(body, want) {
			t.Fatalf("body %q does not contain %q", body, want)
		}
	}
	if strings.Contains(body, "validation:") {
		t.Fatalf("body %q leaks the internal layer name", body)
	}
}

func TestWriteValidationErrorIgnoresNil(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteValidationError(rec, httptest.NewRequest(http.MethodPost, "/api/tasks", nil), nil)
	if rec.Body.Len() != 0 {
		t.Fatalf("a nil error wrote %q; it must write nothing", rec.Body.String())
	}
}

// A 4xx with an empty message is worse than a 500: the status says the request
// was wrong, the code says which rule, and the message — the only part a person
// or an agent reads — is blank. The UI renders exactly that field, so an empty
// string here is an empty red box in the product.
//
// These cases pin the message for every sentinel branch classify has.
func TestClassifyKeepsTheHandlerWrittenMessage(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantCode   string
		wantStatus int
		wantIn     string
	}{
		{
			name:       "bad request names the field and the range",
			err:        fmt.Errorf("%w: progress_pct must be between 0 and 100", ErrBadRequest),
			wantCode:   "bad_request",
			wantStatus: http.StatusBadRequest,
			wantIn:     "progress_pct must be between 0 and 100",
		},
		{
			name:       "bad request carries the accepted vocabulary",
			err:        fmt.Errorf("%w: invalid status %q; must be one of: %s", ErrBadRequest, "levitating", "active, idle"),
			wantCode:   "bad_request",
			wantStatus: http.StatusBadRequest,
			wantIn:     "active, idle",
		},
		{
			name:       "not found keeps the resource detail",
			err:        fmt.Errorf("agent not found: %w", ErrNotFound),
			wantCode:   "not_found",
			wantStatus: http.StatusNotFound,
			wantIn:     "agent not found",
		},
		{
			name:       "conflict classification keeps its full message",
			err:        fmt.Errorf("milestone has 3 open task(s)"),
			wantCode:   "milestone_open_tasks",
			wantStatus: http.StatusConflict,
			wantIn:     "3 open task",
		},
		{
			name:       "unclassified errors still surface their text",
			err:        fmt.Errorf("insert blocker: connection reset"),
			wantCode:   "internal",
			wantStatus: http.StatusInternalServerError,
			wantIn:     "connection reset",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, msg, status := classify(tt.err)
			if code != tt.wantCode {
				t.Errorf("code = %q, want %q", code, tt.wantCode)
			}
			if status != tt.wantStatus {
				t.Errorf("status = %d, want %d", status, tt.wantStatus)
			}
			if msg == "" {
				t.Fatal("message is empty: the caller has a status and a code but nothing to read")
			}
			if !strings.Contains(msg, tt.wantIn) {
				t.Errorf("message %q does not contain %q", msg, tt.wantIn)
			}
		})
	}
}

// The sentinel's own text is transport vocabulary. Repeating it in the message
// tells the reader nothing and shows them the shape of the code instead of the
// problem.
func TestClassifyDropsTheSentinelPrefix(t *testing.T) {
	_, msg, _ := classify(fmt.Errorf("%w: task_id is required", ErrBadRequest))
	if strings.HasPrefix(msg, ErrBadRequest.Error()) {
		t.Fatalf("message %q still carries the sentinel prefix", msg)
	}
	if msg != "task_id is required" {
		t.Fatalf("message = %q, want %q", msg, "task_id is required")
	}
}

// A handler that wraps the sentinel and writes nothing of its own is passing no
// detail to the caller. That is the handler's mistake, not something classify
// should paper over with a blank message — repeating the sentinel's own words
// makes the response self-describing ("400 bad request") instead of returning a
// 400 that says nothing at all.
func TestClassifyFallsBackToTheSentinelTextForABareWrap(t *testing.T) {
	_, msg, _ := classify(fmt.Errorf("%w", ErrBadRequest))
	if msg != ErrBadRequest.Error() {
		t.Fatalf("message = %q, want the sentinel's own text %q", msg, ErrBadRequest.Error())
	}
}
