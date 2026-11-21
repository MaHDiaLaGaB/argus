// Package llm is the one interface every model call in argus goes through
// (ARCHITECTURE.md Sections 5b, 10.3 and 19).
//
// Provider clients (Anthropic, OpenAI, xAI) implement Client with plain HTTPS;
// there is no AI framework. Budgets, pricing, fallbacks and logging wrap a
// Client, so they work the same for every provider. Every place that calls
// Complete must be listed in docs/model-calls.md under its CallSite name.
//
// Requests are checked by Validate before they leave the process, so a
// malformed conversation fails in code, costs nothing and never reaches a provider.
package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/argus-dev/argus/internal/contracts"
)

// Client sends one request to a model and returns its reply.
//
// Implementations must call req.Validate first and return an *Error of kind
// ErrInvalidRequest when it fails, honour ctx cancellation, and fill
// Response.Usage from the provider's own counts (never estimates).
// A Client does not retry or fall back; wrappers do that.
type Client interface {
	Complete(ctx context.Context, req Request) (Response, error)
}

// Request is one model call, in a provider-neutral shape.
type Request struct {
	// Model is the model ref "provider:name" from config/models.yaml.
	Model string
	// CallSite names the code that makes the call, as listed in docs/model-calls.md
	// (for example "agents.writer"). Used for logging, budgets and the E04.07 check.
	CallSite string
	// System is the stable instruction prefix. Untrusted data never goes here.
	System string
	// Messages alternate user, assistant, user, ... and end with a user message.
	Messages []Message
	// Tools the model may call. Each input schema is a JSON Schema object.
	Tools []Tool
	// MaxTokens caps the reply length.
	MaxTokens int
	// Cache says which stable prefix providers should cache.
	Cache CacheHint
	// Meta ties the call to its run and step for tracing, cost and replay.
	Meta Meta
}

// Meta identifies where a call belongs. It is never sent to the provider.
type Meta struct {
	OrgID   string
	RunID   string
	WorkID  string // empty for calls outside a WorkTicket (for example intake)
	StepID  string
	Attempt int
}

// CacheHint asks the provider to cache a stable prefix (ARCHITECTURE 10.3:
// stable prompt prefixes for provider caching). Providers with automatic
// caching may ignore it; it never changes the reply.
type CacheHint struct {
	// Prefix caches tools and the system prompt.
	Prefix bool
	// History also caches the conversation up to the last message. Needs Prefix.
	History bool
}

// Role is who wrote a message.
type Role string

// Message roles. System text goes in Request.System, not in a message.
const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// Message is one turn of the conversation.
type Message struct {
	Role  Role
	Parts []Part
}

// Part is exactly one of: text, a tool call (assistant only) or a tool result (user only).
type Part struct {
	Text       string
	ToolCall   *ToolCall
	ToolResult *ToolResult
}

// ToolCall is the model asking to run a tool.
type ToolCall struct {
	ID    string
	Name  string
	Input json.RawMessage // a JSON object
}

// ToolResult is the output of a tool call, sent back in the next user message.
type ToolResult struct {
	CallID  string
	Content string
	IsError bool
}

// Tool is a tool the model may call.
type Tool struct {
	Name        string
	Description string
	InputSchema json.RawMessage // JSON Schema with "type": "object"
}

// StopReason says why the model stopped.
type StopReason string

// Normalized stop reasons.
const (
	StopEnd       StopReason = "end"        // finished its reply
	StopToolUse   StopReason = "tool_use"   // wants the tool calls in the reply run
	StopMaxTokens StopReason = "max_tokens" // hit Request.MaxTokens; the reply is cut off
	StopRefusal   StopReason = "refusal"    // the provider's safety system declined
	StopOther     StopReason = "other"      // anything else the provider reports
)

// Response is the model's reply.
type Response struct {
	Model      string // model ref that answered (a fallback may differ from the request)
	Parts      []Part // text and tool calls
	StopReason StopReason
	Usage      Usage
	Latency    time.Duration
	RequestID  string // provider request id, for support and tracing
}

// Usage is normalized token usage. InputTokens counts only uncached input, so
// the four counts never overlap and pricing can bill each at its own rate.
type Usage struct {
	InputTokens      int
	OutputTokens     int
	CacheReadTokens  int
	CacheWriteTokens int
}

// TotalInput is every input token, cached or not.
func (u Usage) TotalInput() int { return u.InputTokens + u.CacheReadTokens + u.CacheWriteTokens }

// Add returns the sum of two usages, for totals per step, run or ticket.
func (u Usage) Add(o Usage) Usage {
	return Usage{
		InputTokens:      u.InputTokens + o.InputTokens,
		OutputTokens:     u.OutputTokens + o.OutputTokens,
		CacheReadTokens:  u.CacheReadTokens + o.CacheReadTokens,
		CacheWriteTokens: u.CacheWriteTokens + o.CacheWriteTokens,
	}
}

// Text joins the text parts of the reply.
func (r Response) Text() string {
	var b strings.Builder
	for _, p := range r.Parts {
		b.WriteString(p.Text)
	}
	return b.String()
}

// ToolCalls returns the tool calls in the reply, in order.
func (r Response) ToolCalls() []ToolCall {
	var out []ToolCall
	for _, p := range r.Parts {
		if p.ToolCall != nil {
			out = append(out, *p.ToolCall)
		}
	}
	return out
}

// Message turns the reply into the assistant message to append to the conversation.
func (r Response) Message() Message { return Message{Role: RoleAssistant, Parts: r.Parts} }

// UserText is a user message with one text part.
func UserText(s string) Message { return Message{Role: RoleUser, Parts: []Part{{Text: s}}} }

// AssistantText is an assistant message with one text part.
func AssistantText(s string) Message { return Message{Role: RoleAssistant, Parts: []Part{{Text: s}}} }

var toolNameRE = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

// Validate checks the request in code before it is sent. It returns an *Error
// of kind ErrInvalidRequest listing every problem.
func (r *Request) Validate() error {
	var p []string
	add := func(format string, args ...any) { p = append(p, fmt.Sprintf(format, args...)) }

	if _, _, err := contracts.ParseModelRef(r.Model); err != nil {
		add("model: %v", err)
	}
	if strings.TrimSpace(r.CallSite) == "" {
		add("call_site is required (list it in docs/model-calls.md)")
	}
	for field, v := range map[string]string{"org_id": r.Meta.OrgID, "run_id": r.Meta.RunID, "step_id": r.Meta.StepID} {
		if strings.TrimSpace(v) == "" {
			add("meta.%s is required", field)
		}
	}
	if r.MaxTokens <= 0 {
		add("max_tokens must be > 0")
	}
	if r.Cache.History && !r.Cache.Prefix {
		add("cache.history needs cache.prefix")
	}

	tools := map[string]bool{}
	for i, t := range r.Tools {
		if !toolNameRE.MatchString(t.Name) {
			add("tools[%d]: tool name %q must match %s", i, t.Name, toolNameRE)
		}
		if tools[t.Name] {
			add("tools[%d]: %q listed twice", i, t.Name)
		}
		tools[t.Name] = true
		if strings.TrimSpace(t.Description) == "" {
			add("tools[%d]: description is required", i)
		}
		var schema struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(t.InputSchema, &schema); err != nil {
			add("tools[%d]: input_schema: %v", i, err)
		} else if schema.Type != "object" {
			add(`tools[%d]: input_schema must have "type": "object"`, i)
		}
	}

	if len(r.Messages) == 0 {
		add("messages must not be empty")
	} else {
		if r.Messages[0].Role != RoleUser {
			add("first message must be from the user")
		}
		if r.Messages[len(r.Messages)-1].Role != RoleUser {
			add("last message must be from the user (no assistant prefill)")
		}
	}

	seen := map[string]bool{}    // every tool call id so far
	pending := map[string]bool{} // calls from the last assistant turn not yet answered
	for i, m := range r.Messages {
		at := fmt.Sprintf("messages[%d]", i)
		if m.Role != RoleUser && m.Role != RoleAssistant {
			add("%s: role %q must be user or assistant", at, m.Role)
		}
		if i > 0 && m.Role == r.Messages[i-1].Role {
			add("%s: roles must alternate user and assistant", at)
		}
		if len(m.Parts) == 0 {
			add("%s has no parts", at)
		}
		if m.Role == RoleUser && i > 0 {
			for id := range pending {
				answered := false
				for _, part := range m.Parts {
					if part.ToolResult != nil && part.ToolResult.CallID == id {
						answered = true
					}
				}
				if !answered {
					add("%s: tool call %q has no result", at, id)
				}
			}
			pending = map[string]bool{}
		}
		for j, part := range m.Parts {
			pat := fmt.Sprintf("%s.parts[%d]", at, j)
			kinds := 0
			if part.Text != "" {
				kinds++
			}
			if part.ToolCall != nil {
				kinds++
			}
			if part.ToolResult != nil {
				kinds++
			}
			if kinds != 1 {
				add("%s must have exactly one of text, tool_call, tool_result", pat)
				continue
			}
			switch {
			case part.ToolCall != nil:
				c := part.ToolCall
				if m.Role != RoleAssistant {
					add("%s: tool calls only come from the assistant", pat)
				}
				if !tools[c.Name] {
					add("%s: unknown tool %q", pat, c.Name)
				}
				if c.ID == "" {
					add("%s: tool call id is required", pat)
				} else if seen[c.ID] {
					add("%s: tool call id %q used twice", pat, c.ID)
				}
				seen[c.ID], pending[c.ID] = true, true
				var obj map[string]any
				if json.Unmarshal(c.Input, &obj) != nil || obj == nil {
					add("%s: tool call input must be a JSON object", pat)
				}
			case part.ToolResult != nil:
				if m.Role != RoleUser {
					add("%s: tool results only come from the user", pat)
				}
				if !seen[part.ToolResult.CallID] {
					add("%s: no earlier tool call %q", pat, part.ToolResult.CallID)
				}
			}
		}
	}
	if len(p) > 0 {
		return &Error{Model: r.Model, Kind: ErrInvalidRequest, Err: fmt.Errorf("invalid request: %s", strings.Join(p, "; "))}
	}
	return nil
}
