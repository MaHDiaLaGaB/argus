package contracts

import "sort"

var kinds = map[string]func() Validator{
	"ticket":        func() Validator { return &Ticket{} },
	"workticket":    func() Validator { return &WorkTicket{} },
	"envelope":      func() Validator { return &TaskEnvelope{} },
	"writer-result": func() Validator { return &WriterResult{} },
	"review":        func() Validator { return &ReviewReport{} },
	"gate":          func() Validator { return &GateResult{} },
	"merge-brief":   func() Validator { return &MergeBrief{} },
	"score":         func() Validator { return &ScoreRecord{} },
}

// New returns an empty contract of the given kind.
func New(kind string) (Validator, bool) {
	f, ok := kinds[kind]
	if !ok {
		return nil, false
	}
	return f(), true
}

// Kinds lists every contract kind name, sorted.
func Kinds() []string {
	out := make([]string, 0, len(kinds))
	for k := range kinds {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
