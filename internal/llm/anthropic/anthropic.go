// Package anthropic is the Claude client: the Messages API over plain net/http,
// implementing llm.Client. No SDK, no framework (AGENTS.md rule 3).
//
// It maps the provider-neutral llm.Request to the wire format, places prompt
// cache markers from llm.CacheHint, normalizes usage, passes thinking and other
// unknown blocks back unchanged as llm.Opaque parts, and retries 429, 529 and
// 5xx responses with backoff. It never sends sampling parameters: API version
// 2026-10-21 rejects them.
package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/argus-dev/argus/internal/contracts"
	"github.com/argus-dev/argus/internal/llm"
)

// Wire constants.
const (
	DefaultBaseURL = "https://api.anthropic.com"
	APIVersion     = "2026-10-21"
	provider       = contracts.FamilyAnthropic
)

// Retry tuning.
const (
	baseBackoff   = 500 * time.Millisecond
	maxBackoff    = 8 * time.Second
	maxRetryAfter = 60 * time.Second
	maxBodyBytes  = 32 << 20
)

// Config configures a Client. The API key comes from the secret manager or
// environment of the caller; it is sent only in the x-api-key header and never
// logged or put in an error.
type Config struct {
	APIKey     string
	BaseURL    string       // default DefaultBaseURL
	HTTPClient *http.Client // default: a client with a 10 minute timeout
	// MaxRetries for retryable failures: 0 means the default (2), negative means none.
	MaxRetries int

	sleep func(context.Context, time.Duration) error // tests replace the backoff wait
}

// Client calls the Claude Messages API. It is safe for concurrent use.
type Client struct {
	apiKey     string
	baseURL    string
	http       *http.Client
	maxRetries int
	sleep      func(context.Context, time.Duration) error
}

// New returns a Client, or an error if the configuration is unusable.
func New(cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, errors.New("anthropic: API key is required")
	}
	c := &Client{apiKey: cfg.APIKey, baseURL: strings.TrimRight(cfg.BaseURL, "/"), http: cfg.HTTPClient,
		maxRetries: cfg.MaxRetries, sleep: cfg.sleep}
	if c.baseURL == "" {
		c.baseURL = DefaultBaseURL
	}
	if c.http == nil {
		c.http = &http.Client{Timeout: 10 * time.Minute}
	}
	switch {
	case c.maxRetries == 0:
		c.maxRetries = 2
	case c.maxRetries < 0:
		c.maxRetries = 0
	}
	if c.sleep == nil {
		c.sleep = sleepCtx
	}
	return c, nil
}

// Complete implements llm.Client.
func (c *Client) Complete(ctx context.Context, req llm.Request) (llm.Response, error) {
	if err := req.Validate(); err != nil {
		return llm.Response{}, err
	}
	prov, name, _ := contracts.ParseModelRef(req.Model)
	if prov != provider {
		return llm.Response{}, c.err(req.Model, llm.ErrInvalidRequest, 0, fmt.Errorf("model %q is not an Anthropic model", req.Model))
	}
	wreq := buildRequest(name, req)
	for i, m := range wreq.Messages {
		if len(m.Content) == 0 {
			return llm.Response{}, c.err(req.Model, llm.ErrInvalidRequest, 0,
				fmt.Errorf("messages[%d] is empty after dropping other providers' blocks", i))
		}
	}
	body, err := json.Marshal(wreq)
	if err != nil {
		return llm.Response{}, c.err(req.Model, llm.ErrInvalidRequest, 0, err)
	}

	start := time.Now()
	for attempt := 0; ; attempt++ {
		resp, wait, err := c.send(ctx, req.Model, body)
		if err == nil {
			resp.Latency = time.Since(start)
			return resp, nil
		}
		e, isLLM := errors.AsType[*llm.Error](err)
		if !isLLM || !e.Retryable() || attempt >= c.maxRetries || wait < 0 {
			return llm.Response{}, err
		}
		if wait == 0 {
			wait = backoff(attempt)
		}
		if serr := c.sleep(ctx, wait); serr != nil {
			return llm.Response{}, serr
		}
	}
}

// send makes one HTTP attempt. wait is the server's requested delay before a
// retry: 0 when it gave none, negative when it said not to retry.
func (c *Client) send(ctx context.Context, model string, body []byte) (llm.Response, time.Duration, error) {
	hreq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return llm.Response{}, 0, c.err(model, llm.ErrInvalidRequest, 0, err)
	}
	hreq.Header.Set("content-type", "application/json")
	hreq.Header.Set("x-api-key", c.apiKey)
	hreq.Header.Set("anthropic-version", APIVersion)

	hresp, err := c.http.Do(hreq)
	if err != nil {
		if ctx.Err() != nil {
			return llm.Response{}, 0, ctx.Err()
		}
		if ne, ok := errors.AsType[net.Error](err); ok && ne.Timeout() {
			return llm.Response{}, 0, c.err(model, llm.ErrTimeout, 0, errors.New("request timed out"))
		}
		return llm.Response{}, 0, c.err(model, llm.ErrServer, 0, fmt.Errorf("transport: %w", err))
	}
	defer func() { _ = hresp.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(hresp.Body, maxBodyBytes))
	if err != nil {
		return llm.Response{}, 0, c.err(model, llm.ErrServer, hresp.StatusCode, fmt.Errorf("read body: %w", err))
	}

	if hresp.StatusCode != http.StatusOK {
		kind, msg := classify(hresp.StatusCode, data)
		wait := retryAfter(hresp.Header.Get("retry-after"))
		switch hresp.Header.Get("x-should-retry") {
		case "false":
			wait = -1
		case "true":
			if kind == llm.ErrInvalidRequest {
				kind = llm.ErrServer
			}
		}
		return llm.Response{}, wait, c.err(model, kind, hresp.StatusCode, errors.New(msg))
	}

	resp, err := parseResponse(data)
	if err != nil {
		// The call may already be billed, so an unreadable reply is not retried.
		return llm.Response{}, -1, c.err(model, llm.ErrServer, hresp.StatusCode, err)
	}
	resp.RequestID = hresp.Header.Get("request-id")
	return resp, 0, nil
}

func (c *Client) err(model string, kind llm.ErrorKind, status int, err error) *llm.Error {
	return &llm.Error{Provider: provider, Model: model, Kind: kind, Status: status, Err: err}
}

// classify maps an error response to an error kind and a short message.
func classify(status int, body []byte) (llm.ErrorKind, string) {
	var e struct {
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
			Details struct {
				ErrorCode string `json:"error_code"`
			} `json:"details"`
		} `json:"error"`
	}
	msg := fmt.Sprintf("HTTP %d", status)
	if json.Unmarshal(body, &e) == nil && e.Error.Type != "" {
		msg = e.Error.Type + ": " + truncate(e.Error.Message, 300)
	}
	switch {
	case status == http.StatusTooManyRequests && e.Error.Details.ErrorCode == "enforced_spend_limit_reached":
		return llm.ErrBudget, msg // a spend cap does not lift on retry
	case status == http.StatusTooManyRequests:
		return llm.ErrRateLimited, msg
	case status == http.StatusPaymentRequired:
		return llm.ErrBudget, msg
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return llm.ErrAuth, msg
	case status == 529:
		return llm.ErrOverloaded, msg
	case status >= 500:
		return llm.ErrServer, msg
	default: // 400, 404, 413 and anything else from 4xx
		return llm.ErrInvalidRequest, msg
	}
}

// retryAfter parses a retry-after header in seconds, capped at maxRetryAfter.
func retryAfter(v string) time.Duration {
	secs, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
	if err != nil || secs <= 0 {
		return 0
	}
	return min(time.Duration(secs*float64(time.Second)), maxRetryAfter)
}

// backoff is exponential with jitter: base*2^attempt scaled by [0.5, 1.5), capped.
func backoff(attempt int) time.Duration {
	d := baseBackoff << min(attempt, 10)
	d = time.Duration(float64(d) * (0.5 + rand.Float64())) // #nosec G404 -- jitter, not security
	return max(min(d, maxBackoff), time.Millisecond)
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
