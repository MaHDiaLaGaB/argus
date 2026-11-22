// Package openai speaks the OpenAI Responses API (POST /v1/responses) over
// plain net/http. Calls are stateless (store false): argus keeps the
// conversation and replays OpenAI's own output items unchanged and in order,
// as the API requires for reasoning models.
package openai

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/argus-dev/argus/internal/contracts"
	"github.com/argus-dev/argus/internal/llm"
)

// minOutputTokens is the smallest max_output_tokens the API accepts.
// The cap covers reasoning tokens as well as visible output.
const minOutputTokens = 16

type wireRequest struct {
	Model           string     `json:"model"`
	Store           bool       `json:"store"`
	MaxOutputTokens int        `json:"max_output_tokens"`
	Tools           []wireTool `json:"tools,omitempty"`
	Input           []any      `json:"input"`
}

type wireTool struct {
	Type        string          `json:"type"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
	// Strict is always sent as false: when omitted the API may apply strict
	// validation, which needs schemas in a restricted form.
	Strict bool `json:"strict"`
}

// wireMessage content is a list of inputText, or a plain string for an
// assistant turn written by another provider.
type wireMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type inputText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type functionCall struct {
	Type      string `json:"type"`
	CallID    string `json:"call_id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON encoded as a string
}

type functionCallOutput struct {
	Type   string `json:"type"`
	CallID string `json:"call_id"`
	Output string `json:"output"`
}

// buildRequest maps a validated llm.Request to a request body for provider.
// CacheHint has no effect: OpenAI caches stable prefixes automatically. The
// system prompt is the first developer message, which keeps it in that prefix.
func buildRequest(provider string, r llm.Request) ([]byte, error) {
	invalid := func(err error) error {
		return &llm.Error{Provider: provider, Model: r.Model, Kind: llm.ErrInvalidRequest, Err: err}
	}
	prov, name, err := contracts.ParseModelRef(r.Model)
	if err != nil {
		return nil, invalid(err)
	}
	if prov != provider {
		return nil, invalid(fmt.Errorf("model %q is not a model of %s", r.Model, provider))
	}
	if r.MaxTokens < minOutputTokens {
		return nil, invalid(fmt.Errorf("max_tokens %d is below the API minimum %d", r.MaxTokens, minOutputTokens))
	}

	w := wireRequest{Model: name, MaxOutputTokens: r.MaxTokens}
	for _, t := range r.Tools {
		w.Tools = append(w.Tools, wireTool{Type: "function", Name: t.Name, Description: t.Description, Parameters: t.InputSchema})
	}
	if r.System != "" {
		w.Input = append(w.Input, wireMessage{Role: "developer", Content: []inputText{{"input_text", r.System}}})
	}
	for _, m := range r.Messages {
		var items []any
		if m.Role == llm.RoleAssistant {
			items, err = assistantItems(provider, m.Parts)
		} else {
			items = userItems(m.Parts)
		}
		if err != nil {
			return nil, invalid(err)
		}
		w.Input = append(w.Input, items...)
	}
	body, err := json.Marshal(w)
	if err != nil {
		return nil, invalid(err)
	}
	return body, nil
}

// userItems groups consecutive texts into one message; each tool result is its own item.
func userItems(parts []llm.Part) []any {
	var items []any
	var texts []inputText
	flush := func() {
		if len(texts) > 0 {
			items = append(items, wireMessage{Role: string(llm.RoleUser), Content: texts})
			texts = nil
		}
	}
	for _, p := range parts {
		if p.ToolResult == nil {
			texts = append(texts, inputText{"input_text", p.Text})
			continue
		}
		flush()
		out := p.ToolResult.Content
		if p.ToolResult.IsError { // the API has no error flag: the output must say so
			out = "Tool error: " + out
		}
		items = append(items, functionCallOutput{"function_call_output", p.ToolResult.CallID, out})
	}
	flush()
	return items
}

// assistantItems replays provider's own items unchanged and rebuilds turns
// written by another provider, dropping that provider's opaque blocks.
func assistantItems(provider string, parts []llm.Part) ([]any, error) {
	var items []any
	for _, p := range parts {
		switch {
		case p.Opaque != nil:
			if p.Opaque.Provider == provider {
				items = append(items, p.Opaque.Raw)
			}
		case p.Native != nil && p.Native.Provider == provider:
			items = append(items, p.Native.Raw)
		case p.ToolCall != nil:
			var args bytes.Buffer
			if err := json.Compact(&args, p.ToolCall.Input); err != nil {
				return nil, fmt.Errorf("tool call %q: %w", p.ToolCall.ID, err)
			}
			items = append(items, functionCall{"function_call", p.ToolCall.ID, p.ToolCall.Name, args.String()})
		default:
			items = append(items, wireMessage{Role: string(llm.RoleAssistant), Content: p.Text})
		}
	}
	return items, nil
}
