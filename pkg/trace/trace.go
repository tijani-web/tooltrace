package trace

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

// Call represents one tool invocation recorded in a trace file.
// This struct is the public contract of the trace format — fields must
// not be removed or repurposed across versions without a major bump.
type Call struct {
	SessionID  string          `json:"session_id"`
	Sequence   int64           `json:"sequence"`
	Timestamp  string          `json:"timestamp"`
	Tool       string          `json:"tool"`
	Arguments  json.RawMessage `json:"arguments"`
	Result     json.RawMessage `json:"result"`
	DurationMS int64           `json:"duration_ms"`
	Error      *string         `json:"error"`
}

// Read reads all Call records from a .jsonl trace file in order.
// It returns a clear error (never panics) on malformed JSON in any line.
// An empty file returns an empty slice and no error.
func Read(path string) ([]Call, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("tooltrace: open trace file: %w", err)
	}
	defer f.Close()

	var calls []Call
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var c Call
		if err := json.Unmarshal(line, &c); err != nil {
			return nil, fmt.Errorf("tooltrace: malformed JSON on line %d: %w", lineNum, err)
		}
		calls = append(calls, c)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("tooltrace: reading trace file: %w", err)
	}
	return calls, nil
}
