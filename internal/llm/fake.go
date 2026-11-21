package llm

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"sync"
	"time"
)

// Step is one scripted reply of a Fake: a Response, or an Err, after an optional Delay.
type Step struct {
	Response Response
	Err      error
	Delay    time.Duration
}

// Fake is a Client for tests. It never touches the network and costs $0.
// It validates every request like a real client, records a copy of each call,
// and answers from a script (NewFake) or a function (NewFakeFunc).
// It is safe for concurrent use.
type Fake struct {
	mu      sync.Mutex
	script  []Step
	handler func(context.Context, Request) (Response, error)
	calls   []Request
}

// NewFake returns a Fake that answers calls with steps, in order.
// A call after the last step fails.
func NewFake(steps ...Step) *Fake { return &Fake{script: steps} }

// NewFakeFunc returns a Fake that answers every call with fn.
func NewFakeFunc(fn func(context.Context, Request) (Response, error)) *Fake {
	return &Fake{handler: fn}
}

// Complete implements Client.
func (f *Fake) Complete(ctx context.Context, req Request) (Response, error) {
	f.mu.Lock()
	f.calls = append(f.calls, cloneRequest(req))
	if err := req.Validate(); err != nil {
		f.mu.Unlock()
		return Response{}, err
	}
	if f.handler != nil {
		h := f.handler
		f.mu.Unlock()
		return h(ctx, req)
	}
	if len(f.script) == 0 {
		f.mu.Unlock()
		return Response{}, errors.New("llm fake: no scripted step left")
	}
	step := f.script[0]
	f.script = f.script[1:]
	f.mu.Unlock()

	if step.Delay > 0 {
		t := time.NewTimer(step.Delay)
		defer t.Stop()
		select {
		case <-ctx.Done():
			return Response{}, ctx.Err()
		case <-t.C:
		}
	}
	if err := ctx.Err(); err != nil {
		return Response{}, err
	}
	if step.Err != nil {
		return Response{}, step.Err
	}
	resp := step.Response
	if resp.Model == "" {
		resp.Model = req.Model
	}
	return resp, nil
}

// Calls returns copies of every request received, valid or not.
func (f *Fake) Calls() []Request {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]Request, len(f.calls))
	for i, r := range f.calls {
		out[i] = cloneRequest(r)
	}
	return out
}

// Remaining is the number of scripted steps not yet used.
func (f *Fake) Remaining() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.script)
}

// TextResponse is a reply with one text part that ends the turn.
func TextResponse(text string, u Usage) Response {
	return Response{Parts: []Part{{Text: text}}, StopReason: StopEnd, Usage: u}
}

// ToolCallResponse is a reply that asks for one tool call.
func ToolCallResponse(id, name, inputJSON string, u Usage) Response {
	return Response{
		Parts:      []Part{{ToolCall: &ToolCall{ID: id, Name: name, Input: json.RawMessage(inputJSON)}}},
		StopReason: StopToolUse,
		Usage:      u,
	}
}

// cloneRequest deep-copies the slices of a request so recorded calls cannot change later.
func cloneRequest(r Request) Request {
	r.Tools = slices.Clone(r.Tools)
	for i := range r.Tools {
		r.Tools[i].InputSchema = slices.Clone(r.Tools[i].InputSchema)
	}
	r.Messages = slices.Clone(r.Messages)
	for i := range r.Messages {
		parts := slices.Clone(r.Messages[i].Parts)
		for j := range parts {
			if c := parts[j].ToolCall; c != nil {
				cc := *c
				cc.Input = slices.Clone(c.Input)
				parts[j].ToolCall = &cc
			}
			if res := parts[j].ToolResult; res != nil {
				rc := *res
				parts[j].ToolResult = &rc
			}
		}
		r.Messages[i].Parts = parts
	}
	return r
}
