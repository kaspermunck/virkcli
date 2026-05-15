package virk

import "time"

// Envelope is the JSON wrapper emitted when `--envelope` is set. It carries
// (source, kind, version, fetchedAt) metadata alongside the payload so callers
// composing this CLI's output with other tools can branch deterministically on
// (source, kind) without re-parsing data shapes.
type Envelope struct {
	Source    string    `json:"source"`
	Kind      string    `json:"kind"`
	Version   string    `json:"version"`
	Data      any       `json:"data"`
	FetchedAt time.Time `json:"fetchedAt"`
}

// Wrap returns an Envelope with source="virk" and the given kind + payload.
func Wrap(kind string, data any) Envelope {
	return Envelope{
		Source:    "virk",
		Kind:      kind,
		Version:   "1",
		Data:      data,
		FetchedAt: time.Now().UTC(),
	}
}
