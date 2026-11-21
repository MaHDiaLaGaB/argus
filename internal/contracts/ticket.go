package contracts

import (
	"strings"
	"time"
)

// Ticket is the output of Intake (ARCHITECTURE.md 5.1).
type Ticket struct {
	SchemaVersion string       `json:"schema_version"`
	OrgID         string       `json:"org_id"`
	TicketID      string       `json:"ticket_id"`
	Source        string       `json:"source"`
	SourceRef     string       `json:"source_ref"`
	Kind          string       `json:"kind"`
	Mode          string       `json:"mode"`
	Title         string       `json:"title"`
	Body          string       `json:"body"`
	RepoHint      *string      `json:"repo_hint"`
	Requester     Requester    `json:"requester"`
	Attachments   []Attachment `json:"attachments"`
	CreatedAt     time.Time    `json:"created_at"`
}

// Requester is the person who asked for the work.
type Requester struct {
	UserID  string `json:"user_id"`
	Display string `json:"display"`
}

// Attachment is a file attached to a request.
type Attachment struct {
	URI  string `json:"uri"`
	Mime string `json:"mime"`
}

// Validate checks a Ticket.
func (t *Ticket) Validate() error {
	c := newChecker("Ticket")
	c.schema(t.SchemaVersion)
	c.required(t.OrgID, "org_id")
	c.required(t.TicketID, "ticket_id")
	c.oneOf(t.Source, "source", "github", "slack", "webhook", "form", "cli")
	c.required(t.SourceRef, "source_ref")
	c.oneOf(t.Kind, "kind", "request", "bug", "incident", "project")
	c.oneOf(t.Mode, "mode", "existing", "new_project")
	c.required(t.Title, "title")
	c.check(len(t.Title) <= 200, "title must be at most 200 characters")
	if t.RepoHint != nil {
		c.check(validRepo(*t.RepoHint), "repo_hint %q must look like owner/name", *t.RepoHint)
	}
	c.required(t.Requester.UserID, "requester.user_id")
	for i, a := range t.Attachments {
		c.required(a.URI, "attachments["+itoa(i)+"].uri")
		c.required(a.Mime, "attachments["+itoa(i)+"].mime")
	}
	c.check(!t.CreatedAt.IsZero(), "created_at is required")
	return c.err()
}

// validRepo reports whether s looks like "owner/name".
func validRepo(s string) bool {
	owner, name, ok := strings.Cut(s, "/")
	return ok && owner != "" && name != "" && !strings.ContainsAny(name, "/ ") && !strings.ContainsAny(owner, " ")
}
