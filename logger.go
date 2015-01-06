package marathoner

import (
	"encoding/json"
	"io"
)

// StateLogger is configuration updater that just logs state changes
type StateLogger struct {
	w io.Writer
}

// NewStateLogger creates state logger with specified writer
func NewStateLogger(w io.Writer) StateLogger {
	return StateLogger{w}
}

// Update writes state in json format to writer
func (l StateLogger) Update(s State, r *bool) error {
	return json.NewEncoder(l.w).Encode(s)
}
