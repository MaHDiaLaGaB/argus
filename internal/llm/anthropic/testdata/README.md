Wire fixtures for the Anthropic client tests.

- `*_request.json` are golden request bodies: the client must send exactly this JSON (compared as JSON, not bytes).
- `*_response.json` are response bodies the fake server returns.

They are written by hand from the documented Messages API shape (API version 2026-10-21),
not recorded from live calls. E02.08 adds a recorder; recorded responses replace these then.
No API key or other secret may appear in any fixture.
