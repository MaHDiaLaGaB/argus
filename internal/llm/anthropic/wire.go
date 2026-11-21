package anthropic

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/argus-dev/argus/internal/llm"
)

// Wire types for POST /v1/messages (API version 2026-10-21).

type cacheControl struct {
	Type string `json:"type"`
}

var ephemeral = &cacheControl{Type: "ephemeral"}

type wireRequest struct {
	Model     string        `json:"model"`
	MaxTokens int           `json:"max_tokens"`
	System    []wireBlock   `json:"system,omitempty"`
	Tools     []wireTool    `json:"tools,omitempty"`
	Messages  []wireMessage `json:"messages"`
}

type wireTool struct {
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	InputSchema  json.RawMessage `json:"input_schema"`
	CacheControl *cacheControl   `json:"cache_control,omitempty"`
}

type wireMessage struct {
	Role    string      `json:"role"`
	Content []wireBlock `json:"content"`
}

// wireBlock is one content block. Raw, when set, is sent verbatim instead of the fields.
type wireBlock struct {
	Type         string          `json:"type"`
	Text         string          `json:"text,omitempty"`
	ID           string          `json:"id,omitempty"`
	Name         string          `json:"name,omitempty"`
	Input        json.RawMessage `json:"input,omitempty"`
	ToolUseID    string          `json:"tool_use_id,omitempty"`
	Content      *string         `json:"content,omitempty"`
	IsError      bool            `json:"is_error,omitempty"`
	CacheControl *cacheControl   `json:"cache_control,omitempty"`

	Raw json.RawMessage `json:"-"`
}

// MarshalJSON sends opaque blocks unchanged. Cache markers never land on one:
// they go on the last block of the last message, which is always a user message.
func (b wireBlock) MarshalJSON() ([]byte, error) {
	if b.Raw != nil {
		return b.Raw, nil
	}
	type plain wireBlock
	return json.Marshal(plain(b))
}

// buildRequest maps an llm.Request to the wire format. Opaque blocks from
// other providers are dropped; Validate has already checked the conversation.
func buildRequest(model string, r llm.Request) wireRequest {
	w := wireRequest{Model: model, MaxTokens: r.MaxTokens}
	if r.System != "" {
		w.System = []wireBlock{{Type: "text", Text: r.System}}
	}
	for _, t := range r.Tools {
		w.Tools = append(w.Tools, wireTool{Name: t.Name, Description: t.Description, InputSchema: t.InputSchema})
	}
	for _, m := range r.Messages {
		wm := wireMessage{Role: string(m.Role)}
		for _, p := range m.Parts {
			switch {
			case p.Text != "":
				wm.Content = append(wm.Content, wireBlock{Type: "text", Text: p.Text})
			case p.ToolCall != nil:
				wm.Content = append(wm.Content, wireBlock{Type: "tool_use", ID: p.ToolCall.ID, Name: p.ToolCall.Name, Input: p.ToolCall.Input})
			case p.ToolResult != nil:
				content := p.ToolResult.Content
				wm.Content = append(wm.Content, wireBlock{Type: "tool_result", ToolUseID: p.ToolResult.CallID, Content: &content, IsError: p.ToolResult.IsError})
			case p.Opaque != nil && p.Opaque.Provider == provider:
				wm.Content = append(wm.Content, wireBlock{Raw: p.Opaque.Raw})
			}
		}
		w.Messages = append(w.Messages, wm)
	}

	// Cache markers: the prefix ends at the system prompt, or at the last tool
	// when there is no system prompt; history ends at the last block sent.
	if r.Cache.Prefix {
		switch {
		case len(w.System) > 0:
			w.System[len(w.System)-1].CacheControl = ephemeral
		case len(w.Tools) > 0:
			w.Tools[len(w.Tools)-1].CacheControl = ephemeral
		}
	}
	if r.Cache.History && len(w.Messages) > 0 {
		last := &w.Messages[len(w.Messages)-1]
		if n := len(last.Content); n > 0 {
			last.Content[n-1].CacheControl = ephemeral
		}
	}
	return w
}

type wireResponse struct {
	Model      string            `json:"model"`
	Content    []json.RawMessage `json:"content"`
	StopReason string            `json:"stop_reason"`
	Usage      struct {
		InputTokens              int `json:"input_tokens"`
		OutputTokens             int `json:"output_tokens"`
		CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
		CacheReadInputTokens     int `json:"cache_read_input_tokens"`
	} `json:"usage"`
}

// parseResponse maps a 200 body to an llm.Response. Text and tool_use blocks
// become parts; every other block (thinking, redacted thinking, server tools)
// is kept as an opaque part so it can be sent back unchanged.
func parseResponse(data []byte) (llm.Response, error) {
	var w wireResponse
	if err := json.Unmarshal(data, &w); err != nil {
		return llm.Response{}, fmt.Errorf("decode response: %w", err)
	}
	if w.Model == "" {
		return llm.Response{}, errors.New("decode response: no model in reply")
	}
	resp := llm.Response{
		Model:      provider + ":" + w.Model,
		StopReason: stopReason(w.StopReason),
		Usage: llm.Usage{
			InputTokens:      w.Usage.InputTokens,
			OutputTokens:     w.Usage.OutputTokens,
			CacheReadTokens:  w.Usage.CacheReadInputTokens,
			CacheWriteTokens: w.Usage.CacheCreationInputTokens,
		},
	}
	for _, raw := range w.Content {
		var b struct {
			Type  string          `json:"type"`
			Text  string          `json:"text"`
			ID    string          `json:"id"`
			Name  string          `json:"name"`
			Input json.RawMessage `json:"input"`
		}
		if err := json.Unmarshal(raw, &b); err != nil {
			return llm.Response{}, fmt.Errorf("decode content block: %w", err)
		}
		switch b.Type {
		case "text":
			if b.Text != "" {
				resp.Parts = append(resp.Parts, llm.Part{Text: b.Text})
			}
		case "tool_use":
			resp.Parts = append(resp.Parts, llm.Part{ToolCall: &llm.ToolCall{ID: b.ID, Name: b.Name, Input: b.Input}})
		default:
			resp.Parts = append(resp.Parts, llm.Part{Opaque: &llm.Opaque{Provider: provider, Raw: raw}})
		}
	}
	return resp, nil
}

func stopReason(s string) llm.StopReason {
	switch s {
	case "end_turn", "stop_sequence":
		return llm.StopEnd
	case "tool_use":
		return llm.StopToolUse
	case "max_tokens":
		return llm.StopMaxTokens
	case "refusal":
		return llm.StopRefusal
	default:
		return llm.StopOther
	}
}
