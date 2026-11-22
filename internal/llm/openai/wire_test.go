package openai

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/argus-dev/argus/internal/llm"
)

const prov = "openai"

var readFile = llm.Tool{Name: "read_file", Description: "Read one file in scope.",
	InputSchema: json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"}}}`)}

func native(raw string) *llm.Opaque {
	return &llm.Opaque{Provider: prov, Raw: json.RawMessage(raw)}
}

func base(model string, maxTokens int, msgs ...llm.Message) llm.Request {
	return llm.Request{Model: prov + ":" + model, CallSite: "agents.reviewer", MaxTokens: maxTokens, Messages: msgs,
		Meta: llm.Meta{OrgID: "org_test", RunID: "run_1", StepID: "step_1"}}
}

func call(id, input string) *llm.ToolCall {
	return &llm.ToolCall{ID: id, Name: "read_file", Input: json.RawMessage(input)}
}

func TestBuildRequestGolden(t *testing.T) {
	text := base("gpt-6-luna", 512, llm.Message{Role: llm.RoleUser, Parts: []llm.Part{{Text: "Summarize the PR."}, {Text: "Be brief."}}})
	text.System = "Write one line."
	text.Cache = llm.CacheHint{Prefix: true, History: true} // OpenAI caches automatically: no effect

	// OpenAI's own items are replayed exactly, in order: reasoning, message, call.
	replay := base("gpt-6.1-sol", 4096, llm.UserText("Add a notes endpoint."),
		llm.Message{Role: llm.RoleAssistant, Parts: []llm.Part{
			{Opaque: native(`{"type":"reasoning","id":"rs_1","summary":[],"encrypted_content":"enc"}`)},
			{Text: "Reading.", Native: native(`{"type":"message","id":"msg_1","role":"assistant","status":"completed","phase":"commentary","content":[{"type":"output_text","text":"Reading.","annotations":[]}]}`)},
			{ToolCall: call("call_1", `{"path":"a.go"}`), Native: native(`{"type":"function_call","id":"fc_1","call_id":"call_1","name":"read_file","arguments":"{\"path\":\"a.go\"}","status":"completed"}`)},
		}},
		llm.Message{Role: llm.RoleUser, Parts: []llm.Part{{ToolResult: &llm.ToolResult{CallID: "call_1", Content: "package a"}}, {Text: "Go on."}}})
	replay.Tools = []llm.Tool{readFile}

	// A Claude conversation moved here by a fallback: its thinking and native blocks are
	// dropped, its text and calls are rebuilt.
	claude := &llm.Opaque{Provider: "anthropic", Raw: json.RawMessage(`{"type":"thinking"}`)}
	fallback := base("gpt-6-astra", 4096, llm.UserText("Add a notes endpoint."),
		llm.Message{Role: llm.RoleAssistant, Parts: []llm.Part{
			{Opaque: claude}, {Text: "Reading.", Native: claude},
			{ToolCall: call("toolu_1", `{"path": "a.go"}`)}, {ToolCall: call("toolu_2", `{"path":"b.go"}`)},
		}},
		llm.Message{Role: llm.RoleUser, Parts: []llm.Part{
			{ToolResult: &llm.ToolResult{CallID: "toolu_1"}},
			{ToolResult: &llm.ToolResult{CallID: "toolu_2", Content: "not found", IsError: true}},
		}})
	fallback.Tools = []llm.Tool{readFile}

	for name, req := range map[string]llm.Request{"text": text, "replay": replay, "fallback": fallback} {
		t.Run(name, func(t *testing.T) {
			if err := req.Validate(); err != nil {
				t.Fatal(err)
			}
			body, err := buildRequest(prov, req)
			if err != nil {
				t.Fatal(err)
			}
			golden, err := os.ReadFile(filepath.Join("testdata", name+"_request.json"))
			if err != nil {
				t.Fatal(err)
			}
			var got, want any
			if err := errors.Join(json.Unmarshal(body, &got), json.Unmarshal(golden, &want)); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("got  %v\nwant %v", got, want)
			}
		})
	}
}

func TestBuildRequestRejects(t *testing.T) {
	for name, req := range map[string]llm.Request{
		"below the API minimum": base("gpt-6-luna", minOutputTokens-1, llm.UserText("hi")),
		"other provider":        {Model: "anthropic:claude-haiku-5-5", MaxTokens: 512},
		"bad ref":               {Model: "gpt-6-luna", MaxTokens: 512},
	} {
		_, err := buildRequest(prov, req)
		if e, ok := errors.AsType[*llm.Error](err); !ok || e.Kind != llm.ErrInvalidRequest || e.Provider != prov {
			t.Errorf("%s: want ErrInvalidRequest, got %v", name, err)
		}
	}
}
