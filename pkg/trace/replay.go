package trace

import (
	"encoding/json"
	"fmt"
)

// MockRegistry maps tool names to mock handler functions.
// Each handler receives the recorded arguments and returns a result or error,
// mirroring the real tool's signature. A missing tool name is a reportable
// error — the replay step fails with "no mock registered for tool X".
type MockRegistry map[string]func(args json.RawMessage) (json.RawMessage, error)

// ReplayResult holds the outcome of replaying one Call.
type ReplayResult struct {
	Sequence int64
	Tool     string
	Passed   bool
	// Reason is populated on failure with a human-readable explanation.
	Reason string
}

// Replay re-executes each Call in calls against the provided MockRegistry
// and returns one ReplayResult per Call in the same order.
//
// Comparison logic: a step passes when the recorded outcome (success or
// error) matches the mock outcome. Result payloads are not compared in v1
// (that is v3 diff territory).
func Replay(calls []Call, registry MockRegistry) []ReplayResult {
	results := make([]ReplayResult, 0, len(calls))

	for _, c := range calls {
		res := ReplayResult{
			Sequence: c.Sequence,
			Tool:     c.Tool,
		}

		fn, ok := registry[c.Tool]
		if !ok {
			res.Passed = false
			res.Reason = fmt.Sprintf("no mock registered for tool %q", c.Tool)
			results = append(results, res)
			continue
		}

		_, mockErr := fn(c.Arguments)

		recordedFailed := c.Error != nil
		mockFailed := mockErr != nil

		switch {
		case !recordedFailed && !mockFailed:
			// Both succeeded — pass.
			res.Passed = true
		case recordedFailed && mockFailed:
			// Both failed — pass (outcome matches; error text is not compared in v1).
			res.Passed = true
		case !recordedFailed && mockFailed:
			res.Passed = false
			res.Reason = fmt.Sprintf("expected success, got error: %s", mockErr.Error())
		case recordedFailed && !mockFailed:
			res.Passed = false
			res.Reason = fmt.Sprintf("expected error %q, got success", *c.Error)
		}

		results = append(results, res)
	}

	return results
}
