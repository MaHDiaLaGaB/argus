package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/argus-dev/argus/internal/llm"
)

const testKey = "test-key-not-a-secret"

var readFile = llm.Tool{
	Name:        "read_file",
	Description: "Read one file in scope.",
	InputSchema: json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"}},"required":["path"],"additionalProperties":false}`),
}

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// reply is one canned HTTP answer from the fake server.
type reply struct {
	status  int
	body    string
	headers map[string]string
}

// server is a fake Messages API. It records every request and answers from replies in order.
type server struct {
	t       *testing.T
	mu      sync.Mutex
	replies []reply
	bodies  [][]byte
	headers []http.Header
	srv     *httptest.Server
}

func newServer(t *testing.T, replies ...reply) *server {
	s := &server{t: t, replies: replies}
	s.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/messages" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		s.mu.Lock()
		s.bodies = append(s.bodies, body)
		s.headers = append(s.headers, r.Header.Clone())
		if len(s.replies) == 0 {
			s.mu.Unlock()
			t.Errorf("no reply left for request %d", len(s.bodies))
			w.WriteHeader(http.StatusTeapot)
			return
		}
		rp := s.replies[0]
		s.replies = s.replies[1:]
		s.mu.Unlock()
		for k, v := range rp.headers {
			w.Header().Set(k, v)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(rp.status)
		_, _ = w.Write([]byte(rp.body))
	}))
	t.Cleanup(s.srv.Close)
	return s
}

func (s *server) calls() int { s.mu.Lock(); defer s.mu.Unlock(); return len(s.bodies) }

// sleeps records backoff waits instead of sleeping.
type sleeps struct {
	mu sync.Mutex
	d  []time.Duration
}

func (z *sleeps) sleep(ctx context.Context, d time.Duration) error {
	z.mu.Lock()
	z.d = append(z.d, d)
	z.mu.Unlock()
	return ctx.Err()
}

func newClient(t *testing.T, s *server, z *sleeps) *Client {
	t.Helper()
	c, err := New(Config{APIKey: testKey, BaseURL: s.srv.URL, MaxRetries: 2, sleep: z.sleep})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func ok(body []byte) reply {
	return reply{status: 200, body: string(body), headers: map[string]string{"request-id": "req_test_1"}}
}

func apiError(status int, typ, msg string, extra map[string]string) reply {
	return reply{status: status, body: `{"type":"error","error":{"type":"` + typ + `","message":"` + msg + `"},"request_id":"req_err"}`, headers: extra}
}

func assertJSONEqual(t *testing.T, got, want []byte) {
	t.Helper()
	var g, w any
	if err := json.Unmarshal(got, &g); err != nil {
		t.Fatalf("request is not JSON: %v\n%s", err, got)
	}
	if err := json.Unmarshal(want, &w); err != nil {
		t.Fatalf("golden is not JSON: %v", err)
	}
	gb, _ := json.MarshalIndent(g, "", "  ")
	wb, _ := json.MarshalIndent(w, "", "  ")
	if !bytes.Equal(gb, wb) {
		t.Fatalf("request body differs from golden\n--- got\n%s\n--- want\n%s", gb, wb)
	}
}

func meta() llm.Meta {
	return llm.Meta{OrgID: "org_test", RunID: "run_1", WorkID: "TKT-1.1", StepID: "step_1", Attempt: 1}
}

func textRequest() llm.Request {
	return llm.Request{
		Model: "anthropic:claude-haiku-5-5", CallSite: "github.merge_brief", System: "Write one line.",
		MaxTokens: 512, Cache: llm.CacheHint{Prefix: true}, Meta: meta(),
		Messages: []llm.Message{llm.UserText("Summarize the PR.")},
	}
}

func TestTextRoundTrip(t *testing.T) {
	s := newServer(t, ok(fixture(t, "text_response.json")))
	c := newClient(t, s, &sleeps{})
	resp, err := c.Complete(context.Background(), textRequest())
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, s.bodies[0], fixture(t, "text_request.json"))

	h := s.headers[0]
	if h.Get("x-api-key") != testKey || h.Get("anthropic-version") != APIVersion || h.Get("content-type") != "application/json" {
		t.Fatalf("headers: %v", h)
	}
	if resp.Text() != "Adds a notes endpoint." || resp.StopReason != llm.StopEnd {
		t.Fatalf("reply: %q %s", resp.Text(), resp.StopReason)
	}
	if resp.Model != "anthropic:claude-haiku-5-5" || resp.RequestID != "req_test_1" {
		t.Fatalf("model/request id: %s %s", resp.Model, resp.RequestID)
	}
	want := llm.Usage{InputTokens: 21, CacheWriteTokens: 400, OutputTokens: 9}
	if resp.Usage != want {
		t.Fatalf("usage %+v, want %+v", resp.Usage, want)
	}
	if o := resp.Parts[0].Opaque; o == nil || o.Provider != "anthropic" || !strings.Contains(string(o.Raw), "sig-text-1") {
		t.Fatalf("thinking block must come back as an opaque part, got %+v", resp.Parts[0])
	}
}

// Tool calls round-trip: the reply's thinking, text and tool_use go back unchanged,
// followed by the tool result, and the second reply is parsed.
func TestToolRoundTrip(t *testing.T) {
	s := newServer(t, ok(fixture(t, "tool1_response.json")), ok(fixture(t, "tool2_response.json")))
	c := newClient(t, s, &sleeps{})
	req := llm.Request{
		Model: "anthropic:claude-sonnet-5-5", CallSite: "agents.writer", System: "You are the backend Writer.",
		Tools: []llm.Tool{readFile}, MaxTokens: 4096, Cache: llm.CacheHint{Prefix: true}, Meta: meta(),
		Messages: []llm.Message{llm.UserText("Add a notes endpoint.")},
	}
	r1, err := c.Complete(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, s.bodies[0], fixture(t, "tool1_request.json"))
	calls := r1.ToolCalls()
	if r1.StopReason != llm.StopToolUse || len(calls) != 1 || calls[0].ID != "toolu_01A" || string(calls[0].Input) != `{"path": "src/routes.go"}` {
		t.Fatalf("round 1: %s %+v", r1.StopReason, calls)
	}

	req.Messages = append(req.Messages, r1.Message(), llm.Message{Role: llm.RoleUser, Parts: []llm.Part{
		{ToolResult: &llm.ToolResult{CallID: calls[0].ID, Content: "package routes"}},
	}})
	req.Cache.History = true
	req.Meta.StepID = "step_2"
	r2, err := c.Complete(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, s.bodies[1], fixture(t, "tool2_request.json"))
	if r2.Text() != "The endpoint goes in src/routes.go." || r2.Usage.CacheReadTokens != 360 {
		t.Fatalf("round 2: %q %+v", r2.Text(), r2.Usage)
	}
}

func TestCachePlacement(t *testing.T) {
	cases := []struct {
		name  string
		edit  func(r *llm.Request)
		check func(t *testing.T, body map[string]any)
	}{
		{"no hint, no markers", func(r *llm.Request) { r.Cache = llm.CacheHint{} }, func(t *testing.T, b map[string]any) {
			if strings.Contains(mustJSON(b), "cache_control") {
				t.Fatal("no cache_control expected")
			}
		}},
		{"prefix without system marks last tool", func(r *llm.Request) {
			r.System = ""
			r.Tools = []llm.Tool{readFile}
		}, func(t *testing.T, b map[string]any) {
			tools := b["tools"].([]any)
			if _, ok := tools[len(tools)-1].(map[string]any)["cache_control"]; !ok {
				t.Fatal("last tool must carry cache_control")
			}
			if _, ok := b["system"]; ok {
				t.Fatal("empty system must be omitted")
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newServer(t, ok(fixture(t, "text_response.json")))
			req := textRequest()
			tc.edit(&req)
			if _, err := newClient(t, s, &sleeps{}).Complete(context.Background(), req); err != nil {
				t.Fatal(err)
			}
			var body map[string]any
			if err := json.Unmarshal(s.bodies[0], &body); err != nil {
				t.Fatal(err)
			}
			tc.check(t, body)
		})
	}
}

func mustJSON(v any) string { b, _ := json.Marshal(v); return string(b) }

func TestStopReasons(t *testing.T) {
	for wire, want := range map[string]llm.StopReason{
		"end_turn": llm.StopEnd, "stop_sequence": llm.StopEnd, "tool_use": llm.StopToolUse,
		"max_tokens": llm.StopMaxTokens, "refusal": llm.StopRefusal, "pause_turn": llm.StopOther,
	} {
		body := strings.Replace(string(fixture(t, "text_response.json")), `"end_turn"`, `"`+wire+`"`, 1)
		s := newServer(t, ok([]byte(body)))
		r, err := newClient(t, s, &sleeps{}).Complete(context.Background(), textRequest())
		if err != nil || r.StopReason != want {
			t.Errorf("%s: got %s %v, want %s", wire, r.StopReason, err, want)
		}
	}
}

func TestRetries(t *testing.T) {
	good := ok(fixture(t, "text_response.json"))
	cases := []struct {
		name      string
		replies   []reply
		wantCalls int
		wantKind  llm.ErrorKind // "" = success
		wantSleep []time.Duration
	}{
		{"overloaded then ok", []reply{apiError(529, "overloaded_error", "busy", nil), good}, 2, "", nil},
		{"rate limit honours retry-after", []reply{apiError(429, "rate_limit_error", "slow", map[string]string{"retry-after": "3"}), good}, 2, "",
			[]time.Duration{3 * time.Second}},
		{"server errors exhaust retries", []reply{apiError(500, "api_error", "x", nil), apiError(500, "api_error", "x", nil), apiError(500, "api_error", "x", nil)}, 3, llm.ErrServer, nil},
		{"x-should-retry false stops", []reply{apiError(500, "api_error", "x", map[string]string{"x-should-retry": "false"})}, 1, llm.ErrServer, nil},
		{"spend cap is a budget error, not retried", []reply{{status: 429, body: `{"type":"error","error":{"type":"rate_limit_error","message":"cap","details":{"error_code":"enforced_spend_limit_reached"}}}`}}, 1, llm.ErrBudget, nil},
		{"bad request not retried", []reply{apiError(400, "invalid_request_error", "bad", nil)}, 1, llm.ErrInvalidRequest, nil},
		{"auth not retried", []reply{apiError(401, "authentication_error", "who", nil)}, 1, llm.ErrAuth, nil},
		{"billing is budget", []reply{apiError(402, "billing_error", "pay", nil)}, 1, llm.ErrBudget, nil},
		{"model not found", []reply{apiError(404, "not_found_error", "model", nil)}, 1, llm.ErrInvalidRequest, nil},
		{"garbage 200", []reply{{status: 200, body: "not json"}}, 1, llm.ErrServer, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newServer(t, tc.replies...)
			z := &sleeps{}
			_, err := newClient(t, s, z).Complete(context.Background(), textRequest())
			if s.calls() != tc.wantCalls {
				t.Fatalf("calls = %d, want %d", s.calls(), tc.wantCalls)
			}
			if tc.wantKind == "" {
				if err != nil {
					t.Fatalf("want success, got %v", err)
				}
			} else {
				e, isLLM := errors.AsType[*llm.Error](err)
				if !isLLM || e.Kind != tc.wantKind {
					t.Fatalf("want %s, got %v", tc.wantKind, err)
				}
				if e.Provider != "anthropic" || e.Model != "anthropic:claude-haiku-5-5" {
					t.Fatalf("error must name provider and model: %+v", e)
				}
			}
			if tc.wantSleep != nil && (len(z.d) != len(tc.wantSleep) || z.d[0] != tc.wantSleep[0]) {
				t.Fatalf("sleeps %v, want %v", z.d, tc.wantSleep)
			}
			if err != nil && strings.Contains(err.Error(), testKey) {
				t.Fatal("API key leaked into the error")
			}
		})
	}
}

func TestBackoffGrowsAndCaps(t *testing.T) {
	for attempt, max := range map[int]time.Duration{0: 750 * time.Millisecond, 1: 1500 * time.Millisecond, 3: 6 * time.Second, 10: maxBackoff} {
		for range 20 {
			if d := backoff(attempt); d <= 0 || d > max {
				t.Fatalf("backoff(%d) = %v, want in (0, %v]", attempt, d, max)
			}
		}
	}
}

func TestRetryAfterIsCapped(t *testing.T) {
	s := newServer(t, apiError(429, "rate_limit_error", "slow", map[string]string{"retry-after": "3600"}), ok(fixture(t, "text_response.json")))
	z := &sleeps{}
	if _, err := newClient(t, s, z).Complete(context.Background(), textRequest()); err != nil {
		t.Fatal(err)
	}
	if z.d[0] != maxRetryAfter {
		t.Fatalf("retry-after wait %v, want cap %v", z.d[0], maxRetryAfter)
	}
}

func TestCancelledDuringBackoff(t *testing.T) {
	s := newServer(t, apiError(529, "overloaded_error", "busy", nil))
	ctx, cancel := context.WithCancel(context.Background())
	c, err := New(Config{APIKey: testKey, BaseURL: s.srv.URL, sleep: func(context.Context, time.Duration) error { cancel(); return context.Canceled }})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Complete(ctx, textRequest()); !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}
	if s.calls() != 1 {
		t.Fatalf("calls = %d, want 1", s.calls())
	}
}

func TestTimeoutIsRetryableKind(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { time.Sleep(200 * time.Millisecond) }))
	defer s.Close()
	c, err := New(Config{APIKey: testKey, BaseURL: s.URL, MaxRetries: -1, HTTPClient: &http.Client{Timeout: 20 * time.Millisecond}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.Complete(context.Background(), textRequest())
	if e, ok := errors.AsType[*llm.Error](err); !ok || e.Kind != llm.ErrTimeout {
		t.Fatalf("want ErrTimeout, got %v", err)
	}
}

func TestRejectsBeforeSending(t *testing.T) {
	cases := map[string]func(r *llm.Request){
		"invalid request": func(r *llm.Request) { r.MaxTokens = 0 },
		"other provider":  func(r *llm.Request) { r.Model = "openai:gpt-6-luna" },
		"turn empty after dropping foreign blocks": func(r *llm.Request) {
			r.Messages = []llm.Message{llm.UserText("a"), {Role: llm.RoleAssistant, Parts: []llm.Part{{Opaque: &llm.Opaque{Provider: "openai", Raw: json.RawMessage(`{}`)}}}}, llm.UserText("c")}
		},
	}
	for name, edit := range cases {
		t.Run(name, func(t *testing.T) {
			s := newServer(t)
			req := textRequest()
			edit(&req)
			_, err := newClient(t, s, &sleeps{}).Complete(context.Background(), req)
			if e, ok := errors.AsType[*llm.Error](err); !ok || e.Kind != llm.ErrInvalidRequest {
				t.Fatalf("want ErrInvalidRequest, got %v", err)
			}
			if s.calls() != 0 {
				t.Fatal("an invalid request must never reach the API")
			}
		})
	}
}

// Another provider's opaque blocks are dropped (for example after a cross-family
// fallback), and the rest of the turn is sent.
func TestDropsForeignOpaque(t *testing.T) {
	s := newServer(t, ok(fixture(t, "text_response.json")))
	req := textRequest()
	req.Messages = []llm.Message{llm.UserText("a"), {Role: llm.RoleAssistant, Parts: []llm.Part{
		{Opaque: &llm.Opaque{Provider: "openai", Raw: json.RawMessage(`{"type":"reasoning"}`)}}, {Text: "b"}}}, llm.UserText("c")}
	if _, err := newClient(t, s, &sleeps{}).Complete(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(s.bodies[0]), "reasoning") || !strings.Contains(string(s.bodies[0]), `"text":"b"`) {
		t.Fatalf("foreign block must be dropped and text kept: %s", s.bodies[0])
	}
}

func TestNewNeedsKey(t *testing.T) {
	if _, err := New(Config{}); err == nil {
		t.Fatal("missing API key must be an error")
	}
	c, err := New(Config{APIKey: testKey})
	if err != nil || c.baseURL != DefaultBaseURL || c.maxRetries != 2 {
		t.Fatalf("defaults: %+v %v", c, err)
	}
}

// No fixture may contain anything that looks like a real Anthropic key.
func TestFixturesHaveNoSecrets(t *testing.T) {
	entries, err := os.ReadDir("testdata")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if b := fixture(t, e.Name()); bytes.Contains(b, []byte("sk-ant-")) {
			t.Errorf("%s contains an API key", e.Name())
		}
	}
}

var _ llm.Client = (*Client)(nil)
