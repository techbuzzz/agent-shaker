package mcp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestClaimTaskStatementCarriesOwnershipGuard is a regression guard on the
// shape of the claim statement.
//
// It does NOT prove the claim is race-free — only an integration test against
// a real Postgres, with two concurrent claims, can prove that. What it does
// prove is cheaper and catches the specific regression that matters: someone
// reintroducing the check-then-act form. The original bug read the assignee,
// then updated unconditionally; that version of this statement has no guard in
// the WHERE clause and would fail this test.
//
// If you change the SQL, keep the guard in the statement itself. Moving it
// back into Go code reintroduces the race.
func TestClaimTaskStatementCarriesOwnershipGuard(t *testing.T) {
	stmt := strings.ToLower(claimTaskSQL)

	// The guard: a task owned by someone else must not be claimable.
	if !strings.Contains(stmt, "assigned_to is null or assigned_to = $1") {
		t.Fatal("claim statement lost its ownership guard; a claim would overwrite another agent's work")
	}

	// Terminal states must not be resurrectable by a late heartbeat.
	for _, terminal := range []string{"done", "cancelled"} {
		if !strings.Contains(stmt, terminal) {
			t.Fatalf("claim statement does not exclude terminal state %q", terminal)
		}
	}

	// A bare `WHERE id = $2` with nothing after it is the check-then-act bug in
	// its most compact form.
	trimmed := strings.TrimSpace(stmt)
	if strings.HasSuffix(trimmed, "where id = $2") {
		t.Fatal("claim statement filters on id alone; the ownership guard was removed")
	}

	// RETURNING is what lets the caller distinguish a real claim from a refusal
	// without a second round trip, and it supplies project_id for the live
	// update broadcast.
	if !strings.Contains(stmt, "returning") {
		t.Fatal("claim statement lost its RETURNING clause")
	}
}

// TestCompleteTaskStatementMirrorsClaim guards the invariant that one agent
// owns a task on the way out as well as on the way in.
//
// claim_task was made atomic; complete_task still read the assignee and then
// updated unconditionally, which left a window where a task could be
// reassigned and the previous owner would still mark it done. These statements
// must keep the same shape.
func TestCompleteTaskStatementMirrorsClaim(t *testing.T) {
	claim := strings.ToLower(claimTaskSQL)
	complete := strings.ToLower(completeTaskSQL)

	for _, guard := range []string{
		"assigned_to is null or assigned_to = $1",
		"not in ('done', 'cancelled')",
		"$3::uuid is null or project_id = $3",
	} {
		if !strings.Contains(claim, guard) {
			t.Errorf("claim statement lost the guard %q", guard)
		}
		if !strings.Contains(complete, guard) {
			t.Errorf("complete statement lost the guard %q", guard)
		}
	}

	if !strings.Contains(complete, "returning") {
		t.Error("complete statement has no RETURNING clause, so a refusal is indistinguishable from a write")
	}
	if strings.HasSuffix(strings.TrimSpace(complete), "where id = $2") {
		t.Error("complete statement filters on id alone; the ownership guard was removed")
	}
}

// TestNullableProjectID covers the scoping argument: an unbound connection is
// deliberately unscoped, a bound one is constrained, and a malformed id is an
// error rather than a silent wildcard.
func TestNullableProjectID(t *testing.T) {
	got, err := nullableProjectID("")
	if err != nil {
		t.Fatalf("empty project should be accepted: %v", err)
	}
	if got != nil {
		t.Fatalf("empty project should yield NULL, got %v", got)
	}

	valid := "6f1b7d3e-0a2c-4f5e-9b1a-2c3d4e5f6a7b"
	got, err = nullableProjectID(valid)
	if err != nil {
		t.Fatalf("valid uuid rejected: %v", err)
	}
	if got == nil {
		t.Fatal("valid uuid yielded NULL, which would silently unscope the statement")
	}

	if _, err := nullableProjectID("not-a-uuid"); err == nil {
		t.Fatal("a malformed project_id was accepted; that turns a scoping guarantee into a wildcard")
	}
}

// TestNegotiateProtocolVersion pins the negotiation rules.
func TestNegotiateProtocolVersion(t *testing.T) {
	tests := []struct {
		name    string
		header  string
		want    string
		wantErr bool
	}{
		{name: "absent header defaults", header: "", want: mcpProtocolVersion},
		{name: "legacy revision accepted", header: "2024-11-05", want: "2024-11-05"},
		{name: "newer session revision accepted", header: "2025-06-18", want: "2025-06-18"},
		{name: "latest supported revision accepted", header: "2025-11-25", want: "2025-11-25"},
		// The stateless core is deliberately not implemented. Silently
		// downgrading a client that asked for it would leave it waiting for
		// semantics this server does not have.
		{name: "stateless core refused", header: "2026-07-28", wantErr: true},
		{name: "nonsense refused", header: "banana", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/mcp", nil)
			if tc.header != "" {
				r.Header.Set(protocolVersionHeader, tc.header)
			}

			got, err := negotiateProtocolVersion(r)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected refusal for %q, got %q", tc.header, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tc.header, err)
			}
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// TestSupportedProtocolVersionsExcludesStatelessCore documents the deliberate
// boundary. If a future change adds 2026-07-28 to the accepted set, this test
// is the reminder that the transport has to be stateless first.
func TestSupportedProtocolVersionsExcludesStatelessCore(t *testing.T) {
	if supportsProtocolVersion("2026-07-28") {
		t.Fatal("2026-07-28 accepted without the stateless transport; either implement it or keep refusing it")
	}
	for _, v := range supportedProtocolVersions {
		if !supportsProtocolVersion(v) {
			t.Fatalf("declared revision %q is not accepted by supportsProtocolVersion", v)
		}
	}
}

// TestSupportedProtocolVersionsIsACopy protects the package-level slice from
// being mutated by a caller that ranges over it.
func TestSupportedProtocolVersionsIsACopy(t *testing.T) {
	got := SupportedProtocolVersions()
	if len(got) == 0 {
		t.Fatal("no supported versions reported")
	}
	got[0] = "mutated"
	if supportedProtocolVersions[0] == "mutated" {
		t.Fatal("SupportedProtocolVersions returned the backing array; callers can corrupt the package state")
	}
}

// TestUnsupportedVersionResponseIsMachineReadable checks that a refusal carries
// both the HTTP status and the code a client needs to recover without parsing
// prose.
func TestUnsupportedVersionResponseIsMachineReadable(t *testing.T) {
	w := httptest.NewRecorder()
	writeUnsupportedVersion(w, "2026-07-28")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("got status %d, want %d", w.Code, http.StatusBadRequest)
	}

	var body struct {
		JSONRPC string `json:"jsonrpc"`
		Error   struct {
			Code    int `json:"code"`
			Message string
			Data    struct {
				Requested         string   `json:"requested"`
				SupportedVersions []string `json:"supportedVersions"`
			} `json:"data"`
		} `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("refusal body is not JSON: %v (%s)", err, w.Body.String())
	}

	if body.Error.Code != errUnsupportedVersion {
		t.Fatalf("got code %d, want %d", body.Error.Code, errUnsupportedVersion)
	}
	if len(body.Error.Data.SupportedVersions) == 0 {
		t.Fatal("refusal does not list the supported versions; a client cannot recover")
	}
	for _, v := range body.Error.Data.SupportedVersions {
		if !supportsProtocolVersion(v) {
			t.Fatalf("refusal advertises unsupported version %q", v)
		}
	}
}

// TestNegotiatedVersionFallsBackToDefault covers the zero-valued context that
// the legacy SSE transport and direct unit tests produce.
func TestNegotiatedVersionFallsBackToDefault(t *testing.T) {
	if got := (MCPContext{}).negotiatedVersion(); got != mcpProtocolVersion {
		t.Fatalf("got %q, want %q", got, mcpProtocolVersion)
	}
	if got := (MCPContext{ProtocolVersion: "2025-06-18"}).negotiatedVersion(); got != "2025-06-18" {
		t.Fatalf("got %q, want %q", got, "2025-06-18")
	}
}

// TestServerDiscoverReportsCapabilitiesAndVersions covers the stateless
// discovery method: a client must be able to learn the accepted revisions
// without opening a connection state.
func TestServerDiscoverReportsCapabilitiesAndVersions(t *testing.T) {
	h := &MCPHandler{}

	result, rpcErr := h.handleServerDiscover(MCPContext{ProtocolVersion: "2025-06-18"})
	if rpcErr != nil {
		t.Fatalf("unexpected error: %v", rpcErr)
	}

	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("result is not serialisable: %v", err)
	}
	var decoded struct {
		ProtocolVersion   string   `json:"protocolVersion"`
		SupportedVersions []string `json:"supportedVersions"`
		DefaultVersion    string   `json:"defaultVersion"`
		Capabilities      struct {
			Tools *struct{} `json:"tools"`
		} `json:"capabilities"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("decoding discover result: %v", err)
	}

	if decoded.ProtocolVersion != "2025-06-18" {
		t.Fatalf("discover reported %q, want the negotiated 2025-06-18", decoded.ProtocolVersion)
	}
	if decoded.DefaultVersion != mcpProtocolVersion {
		t.Fatalf("discover default %q, want %q", decoded.DefaultVersion, mcpProtocolVersion)
	}
	if len(decoded.SupportedVersions) != len(supportedProtocolVersions) {
		t.Fatalf("discover listed %d versions, want %d", len(decoded.SupportedVersions), len(supportedProtocolVersions))
	}
	if decoded.Capabilities.Tools == nil {
		t.Fatal("discover omitted the tools capability")
	}
}

// TestInitializeEchoesNegotiatedVersion covers the handshake agreeing with the
// transport: a client that names a supported revision gets it confirmed.
func TestInitializeEchoesNegotiatedVersion(t *testing.T) {
	h := &MCPHandler{}

	// Header already negotiated 2025-06-18; the handshake agrees.
	result, rpcErr := h.handleInitialize(
		json.RawMessage(`{"protocolVersion":"2025-06-18","clientInfo":{"name":"probe"}}`),
		MCPContext{ProtocolVersion: "2025-06-18"},
	)
	if rpcErr != nil {
		t.Fatalf("unexpected error: %v", rpcErr)
	}
	got := result.(InitializeResult)
	if got.ProtocolVersion != "2025-06-18" {
		t.Fatalf("initialize echoed %q, want 2025-06-18", got.ProtocolVersion)
	}

	// The handshake may choose a revision only when the transport did not
	// already settle one. A client that sets the header has been through
	// negotiation, and letting the body override it would make initialize,
	// server/discover and the tool-call span disagree about one request.
	result, rpcErr = h.handleInitialize(
		json.RawMessage(`{"protocolVersion":"2025-03-26"}`),
		MCPContext{},
	)
	if rpcErr != nil {
		t.Fatalf("unexpected error: %v", rpcErr)
	}
	got = result.(InitializeResult)
	if got.ProtocolVersion != "2025-03-26" {
		t.Fatalf("initialize echoed %q, want 2025-03-26", got.ProtocolVersion)
	}

	// Header present: the handshake cannot override it.
	result, rpcErr = h.handleInitialize(
		json.RawMessage(`{"protocolVersion":"2025-03-26"}`),
		MCPContext{ProtocolVersion: "2025-06-18", VersionFromHeader: true},
	)
	if rpcErr != nil {
		t.Fatalf("unexpected error: %v", rpcErr)
	}
	got = result.(InitializeResult)
	if got.ProtocolVersion != "2025-06-18" {
		t.Fatalf("initialize echoed %q; the header is authoritative and it said 2025-06-18", got.ProtocolVersion)
	}

	// Handshake asks for the unimplemented stateless core with no header:
	// the default wins rather than failing the handshake.
	result, rpcErr = h.handleInitialize(
		json.RawMessage(`{"protocolVersion":"2026-07-28"}`),
		MCPContext{},
	)
	if rpcErr != nil {
		t.Fatalf("unexpected error: %v", rpcErr)
	}
	got = result.(InitializeResult)
	if got.ProtocolVersion != mcpProtocolVersion {
		t.Fatalf("initialize echoed %q for an unsupported request; want the default %q",
			got.ProtocolVersion, mcpProtocolVersion)
	}
}
