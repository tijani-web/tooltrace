package trace

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// Writer holds an open trace file and serialises concurrent writes with a mutex.
//
// Concurrency guarantee: Writer is safe for concurrent use from multiple
// goroutines within the same process. Cross-process concurrent writes to the
// same file are explicitly unsupported in v1 — sequence numbers will not be
// correct across processes, and no file-level locking is applied.
type Writer struct {
	mu       sync.Mutex
	f        *os.File
	session  string
	sequence int64
}

// NewWriter opens (or creates) the trace file at path in append-only mode and
// returns a Writer ready to record calls for the given sessionID.
func NewWriter(path, sessionID string) (*Writer, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("tooltrace: open writer: %w", err)
	}
	return &Writer{f: f, session: sessionID}, nil
}

// Close flushes and closes the underlying file. Call this via defer after
// NewWriter succeeds.
func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.f.Close()
}

// Record wraps a single tool invocation: it starts a timer, calls fn, stops
// the timer, then appends one Call line to the trace file.
//
// On success fn must return (result JSON, nil).
// On failure fn must return (nil, error) — the error string is stored in the
// Call and result is left null. Either way the Call is recorded; failed calls
// are never dropped.
func Record(w *Writer, tool string, args json.RawMessage, fn func() (json.RawMessage, error)) error {
	start := time.Now()
	result, fnErr := fn()
	durationMS := time.Since(start).Milliseconds()

	w.mu.Lock()
	defer w.mu.Unlock()

	seq := w.sequence
	w.sequence++

	c := Call{
		SessionID:  w.session,
		Sequence:   seq,
		Timestamp:  start.UTC().Format(time.RFC3339),
		Tool:       tool,
		Arguments:  args,
		DurationMS: durationMS,
	}

	if fnErr != nil {
		errStr := fnErr.Error()
		c.Error = &errStr
		c.Result = nil
	} else {
		c.Error = nil
		c.Result = result
	}

	line, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("tooltrace: marshal call: %w", err)
	}
	line = append(line, '\n')

	if _, err := w.f.Write(line); err != nil {
		return fmt.Errorf("tooltrace: write call: %w", err)
	}
	return nil
}
