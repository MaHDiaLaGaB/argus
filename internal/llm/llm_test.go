package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

var readFile = Tool{
	Name:        "read_file",
	Description: "Read one file in scope.",
	InputSchema: json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"}},"required":["path"],"additionalProperties":false}`),
}

// validRequest is a two-turn tool-use conversation: user asks, model calls a tool, user returns the result.
func validRequest() Request {
	return Request{
		Model:     "anthropic:claude-sonnet-5-5",
		CallSite:  "agents.writer",
		System:    "You are the backend Writer.",
		Tools:     []Tool{readFile},
		MaxTokens: 4096,
		Cache:     CacheHint{Prefix: true},
		Meta:      Meta{OrgID: "org_test", RunID: "run_1", WorkID: "TKT-1.1", StepID: "step_3", Attempt: 1},
		Messages: []Message{
			UserText("Add a notes endpoint."),
			{Role: RoleAssistant, Parts: []Part{
				{Text: "I will read the routes first."},
				{ToolCall: &ToolCall{ID: "call_1", Name: "read_file", Input: json.RawMessage(`{"path":"src/routes.go"}`)}},
			}},
			{Role: RoleUser, Parts: []Part{{ToolResult: &ToolResult{CallID: "call_1", Content: "package routes"}}}},
		},
	}
}

func TestRequestValidate(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(r *Request)
		want   string
	}{
		{"valid", func(r *Request) {}, ""},
		{"bad model ref", func(r *Request) { r.Model = "claude-sonnet" }, "provider:name"},
		{"no call site", func(r *Request) { r.CallSite = "" }, "call_site is required"},
		{"no run id", func(r *Request) { r.Meta.RunID = "" }, "meta.run_id is required"},
		{"no step id", func(r *Request) { r.Meta.StepID = "" }, "meta.step_id is required"},
		{"no org id", func(r *Request) { r.Meta.OrgID = "" }, "meta.org_id is required"},
		{"zero max tokens", func(r *Request) { r.MaxTokens = 0 }, "max_tokens must be > 0"},
		{"no messages", func(r *Request) { r.Messages = nil }, "messages must not be empty"},
		{"starts with assistant", func(r *Request) { r.Messages = r.Messages[1:] }, "first message must be from the user"},
		{"bad role", func(r *Request) { r.Messages[0].Role = "system" }, `role "system"`},
		{"same role twice", func(r *Request) {
			r.Messages = append(r.Messages, UserText("again"))
		}, "roles must alternate"},
		{"empty message", func(r *Request) { r.Messages[0].Parts = nil }, "messages[0] has no parts"},
		{"part with two kinds", func(r *Request) {
			r.Messages[1].Parts[0].ToolCall = &ToolCall{ID: "x", Name: "read_file", Input: json.RawMessage(`{}`)}
		}, "exactly one of text, tool_call, tool_result"},
		{"empty part", func(r *Request) { r.Messages[0].Parts = []Part{{}} }, "exactly one of text, tool_call, tool_result"},
		{"tool call from user", func(r *Request) {
			r.Messages[0].Parts = append(r.Messages[0].Parts, Part{ToolCall: &ToolCall{ID: "u", Name: "read_file", Input: json.RawMessage(`{}`)}})
		}, "tool calls only come from the assistant"},
		{"tool result from assistant", func(r *Request) {
			r.Messages[1].Parts = append(r.Messages[1].Parts, Part{ToolResult: &ToolResult{CallID: "call_1"}})
		}, "tool results only come from the user"},
		{"unknown tool called", func(r *Request) { r.Messages[1].Parts[1].ToolCall.Name = "rm_rf" }, `unknown tool "rm_rf"`},
		{"tool input not an object", func(r *Request) { r.Messages[1].Parts[1].ToolCall.Input = json.RawMessage(`"x"`) }, "input must be a JSON object"},
		{"duplicate call id", func(r *Request) {
			r.Messages[1].Parts = append(r.Messages[1].Parts, Part{ToolCall: &ToolCall{ID: "call_1", Name: "read_file", Input: json.RawMessage(`{}`)}})
		}, `tool call id "call_1" used twice`},
		{"result for unknown call", func(r *Request) { r.Messages[2].Parts[0].ToolResult.CallID = "call_9" }, `no earlier tool call "call_9"`},
		{"ends with assistant", func(r *Request) { r.Messages = r.Messages[:2] }, "last message must be from the user"},
		{"unanswered tool call", func(r *Request) { r.Messages[2] = UserText("never mind") }, `tool call "call_1" has no result`},
		{"bad tool name", func(r *Request) { r.Tools[0].Name = "read file" }, "tool name"},
		{"duplicate tool", func(r *Request) { r.Tools = append(r.Tools, readFile) }, "listed twice"},
		{"no tool description", func(r *Request) { r.Tools[0].Description = "" }, "description is required"},
		{"schema not json", func(r *Request) { r.Tools[0].InputSchema = json.RawMessage(`{`) }, "input_schema"},
		{"schema not object type", func(r *Request) { r.Tools[0].InputSchema = json.RawMessage(`{"type":"string"}`) }, `"type": "object"`},
		{"cache history without prefix", func(r *Request) { r.Cache = CacheHint{History: true} }, "cache.history needs cache.prefix"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := validRequest()
			r.Tools = append([]Tool(nil), r.Tools...)
			tc.mutate(&r)
			err := r.Validate()
			if tc.want == "" {
				if err != nil {
					t.Fatalf("expected valid, got %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %v, want it to contain %q", err, tc.want)
			}
		})
	}
}

func TestUsage(t *testing.T) {
	u := Usage{InputTokens: 100, CacheReadTokens: 900, CacheWriteTokens: 50, OutputTokens: 20}
	if got := u.TotalInput(); got != 1050 {
		t.Fatalf("TotalInput = %d, want 1050", got)
	}
	sum := u.Add(Usage{InputTokens: 1, OutputTokens: 2, CacheReadTokens: 3, CacheWriteTokens: 4})
	if sum != (Usage{InputTokens: 101, OutputTokens: 22, CacheReadTokens: 903, CacheWriteTokens: 54}) {
		t.Fatalf("Add = %+v", sum)
	}
}

func TestResponseHelpers(t *testing.T) {
	r := Response{Parts: []Part{
		{Text: "Reading "},
		{ToolCall: &ToolCall{ID: "c1", Name: "read_file", Input: json.RawMessage(`{"path":"a"}`)}},
		{Text: "now."},
	}, StopReason: StopToolUse}
	if r.Text() != "Reading now." {
		t.Fatalf("Text = %q", r.Text())
	}
	if calls := r.ToolCalls(); len(calls) != 1 || calls[0].Name != "read_file" {
		t.Fatalf("ToolCalls = %+v", calls)
	}
	if m := r.Message(); m.Role != RoleAssistant || len(m.Parts) != 3 {
		t.Fatalf("Message = %+v", m)
	}
}

func TestErrorKinds(t *testing.T) {
	cases := []struct {
		kind      ErrorKind
		retryable bool
	}{
		{ErrRateLimited, true}, {ErrServer, true}, {ErrTimeout, true}, {ErrOverloaded, true},
		{ErrInvalidRequest, false}, {ErrAuth, false}, {ErrBudget, false}, {ErrRefused, false},
	}
	for _, tc := range cases {
		e := &Error{Provider: "anthropic", Model: "anthropic:claude-haiku-5-5", Kind: tc.kind, Status: 429, Err: errors.New("boom")}
		if e.Retryable() != tc.retryable {
			t.Errorf("%s retryable = %v, want %v", tc.kind, e.Retryable(), tc.retryable)
		}
		wrapped := fmt.Errorf("step 3: %w", e)
		if IsRetryable(wrapped) != tc.retryable {
			t.Errorf("IsRetryable(wrapped %s) wrong", tc.kind)
		}
		if !errors.Is(wrapped, e.Err) {
			t.Errorf("%s: Unwrap does not reach the cause", tc.kind)
		}
	}
	if IsRetryable(errors.New("plain")) {
		t.Error("a plain error is not retryable")
	}
	if !IsRetryable(context.DeadlineExceeded) {
		t.Error("a deadline is retryable")
	}
	if IsRetryable(context.Canceled) {
		t.Error("cancellation is not retryable")
	}
}

func TestFakeScript(t *testing.T) {
	f := NewFake(
		Step{Response: ToolCallResponse("call_1", "read_file", `{"path":"src/routes.go"}`, Usage{InputTokens: 10, OutputTokens: 5})},
		Step{Err: &Error{Kind: ErrRateLimited, Err: errors.New("slow down")}},
		Step{Response: TextResponse("Done.", Usage{InputTokens: 12, OutputTokens: 3})},
	)
	ctx := context.Background()
	req := validRequest()

	r1, err := f.Complete(ctx, req)
	if err != nil || r1.StopReason != StopToolUse || r1.ToolCalls()[0].ID != "call_1" {
		t.Fatalf("step 1: %+v %v", r1, err)
	}
	if r1.Model != req.Model {
		t.Fatalf("fake must echo the requested model, got %q", r1.Model)
	}
	if _, err := f.Complete(ctx, req); !IsRetryable(err) {
		t.Fatalf("step 2 should be a retryable error, got %v", err)
	}
	r3, err := f.Complete(ctx, req)
	if err != nil || r3.Text() != "Done." || r3.StopReason != StopEnd {
		t.Fatalf("step 3: %+v %v", r3, err)
	}
	if _, err := f.Complete(ctx, req); err == nil || !strings.Contains(err.Error(), "no scripted step left") {
		t.Fatalf("exhausted script should fail, got %v", err)
	}
	if n := len(f.Calls()); n != 4 {
		t.Fatalf("recorded %d calls, want 4", n)
	}
	if f.Remaining() != 0 {
		t.Fatalf("remaining = %d", f.Remaining())
	}
}

func TestFakeRejectsInvalidRequest(t *testing.T) {
	f := NewFake(Step{Response: TextResponse("never", Usage{})})
	req := validRequest()
	req.MaxTokens = 0
	_, err := f.Complete(context.Background(), req)
	var e *Error
	if !errors.As(err, &e) || e.Kind != ErrInvalidRequest {
		t.Fatalf("want ErrInvalidRequest, got %v", err)
	}
	if f.Remaining() != 1 {
		t.Fatal("an invalid request must not consume a scripted step")
	}
}

func TestFakeHonoursContext(t *testing.T) {
	f := NewFake(Step{Response: TextResponse("late", Usage{}), Delay: time.Second})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if _, err := f.Complete(ctx, validRequest()); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("want deadline exceeded, got %v", err)
	}
}

// Recorded calls are copies: later changes by the caller do not rewrite history.
func TestFakeRecordsCopies(t *testing.T) {
	f := NewFake(Step{Response: TextResponse("ok", Usage{})})
	req := validRequest()
	if _, err := f.Complete(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	req.Messages[0].Parts[0].Text = "changed"
	req.Tools[0].Name = "changed"
	got := f.Calls()[0]
	if got.Messages[0].Parts[0].Text != "Add a notes endpoint." || got.Tools[0].Name != "read_file" {
		t.Fatal("recorded request was changed through the caller's slices")
	}
}

func TestFakeHandler(t *testing.T) {
	f := NewFakeFunc(func(_ context.Context, r Request) (Response, error) {
		return TextResponse("echo: "+r.Messages[0].Parts[0].Text, Usage{}), nil
	})
	r, err := f.Complete(context.Background(), validRequest())
	if err != nil || r.Text() != "echo: Add a notes endpoint." {
		t.Fatalf("%+v %v", r, err)
	}
}

// Fake satisfies Client.
var _ Client = (*Fake)(nil)
