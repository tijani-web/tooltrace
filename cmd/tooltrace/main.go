package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/ToolTraceHQ/tooltrace/pkg/trace"
)

const usage = `tooltrace — record and replay LLM tool calls

Usage:
  tooltrace replay --mock <mocks.json> <trace.jsonl>

Commands:
  replay    Re-execute a recorded trace against a mock registry and
            report pass/fail per step.

Flags for replay:
  --mock    Path to a JSON file mapping tool names to mock responses.
            Each entry must have exactly one of:
              "result": <json>   — mock returns this as a success
              "error":  "string" — mock returns this as a failure

Note: recording is done via the Go library (pkg/trace), not via this
binary. See the examples/ directory for usage patterns.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "replay":
		if err := runReplay(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", os.Args[1], usage)
		os.Exit(1)
	}
}

// mockEntry is the discriminated union used in mocks.json.
// Exactly one of Result or Error must be set; both set is invalid.
type mockEntry struct {
	Result json.RawMessage `json:"result"`
	Error  *string         `json:"error"`
}

func runReplay(args []string) error {
	fs := flag.NewFlagSet("replay", flag.ContinueOnError)
	mockPath := fs.String("mock", "", "path to mocks.json (required)")
	fs.Usage = func() { fmt.Fprint(os.Stderr, usage) }

	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return errors.New("replay requires a trace file argument: tooltrace replay --mock <mocks.json> <trace.jsonl>")
	}
	if *mockPath == "" {
		return errors.New("--mock flag is required")
	}

	tracePath := fs.Arg(0)

	// --- load trace ---
	calls, err := trace.Read(tracePath)
	if err != nil {
		return fmt.Errorf("reading trace: %w", err)
	}
	if len(calls) == 0 {
		fmt.Println("trace file is empty — nothing to replay")
		return nil
	}

	// --- load mocks ---
	registry, err := loadMockRegistry(*mockPath)
	if err != nil {
		return fmt.Errorf("loading mock file: %w", err)
	}

	// --- replay ---
	results := trace.Replay(calls, registry)

	passed, failed := 0, 0
	for _, r := range results {
		if r.Passed {
			passed++
			fmt.Printf("[PASS] step %d — %s\n", r.Sequence, r.Tool)
		} else {
			failed++
			fmt.Printf("[FAIL] step %d — %s: %s\n", r.Sequence, r.Tool, r.Reason)
		}
	}

	fmt.Printf("\n%d passed, %d failed\n", passed, failed)
	if failed > 0 {
		os.Exit(1)
	}
	return nil
}

// loadMockRegistry parses a mocks.json file into a MockRegistry.
// Each entry must have exactly one of "result" or "error".
func loadMockRegistry(path string) (trace.MockRegistry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("open mocks file: %w", err)
	}

	var raw map[string]mockEntry
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse mocks file: %w", err)
	}

	registry := make(trace.MockRegistry, len(raw))
	for toolName, entry := range raw {
		// Validation: both fields set is an error.
		hasResult := len(entry.Result) > 0 && string(entry.Result) != "null"
		hasError := entry.Error != nil

		if hasResult && hasError {
			return nil, fmt.Errorf(
				"mock entry for %q has both 'result' and 'error' — exactly one must be set",
				toolName,
			)
		}
		if !hasResult && !hasError {
			return nil, fmt.Errorf(
				"mock entry for %q has neither 'result' nor 'error' — exactly one must be set",
				toolName,
			)
		}

		// Capture loop variable for the closure.
		e := entry
		name := toolName
		_ = name
		registry[toolName] = func(args json.RawMessage) (json.RawMessage, error) {
			if e.Error != nil {
				return nil, errors.New(*e.Error)
			}
			return e.Result, nil
		}
	}
	return registry, nil
}
