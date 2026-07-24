// Package main demonstrates tooltrace library usage with an Anthropic-style
// tool-calling loop (Claude tool use). No live LLM connection is required —
// tool call boundaries and responses are simulated inline.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/ToolTraceHQ/tooltrace/pkg/trace"
)

func main() {
	tracePath := "anthropic_example.jsonl"
	sessionID := "sess_anthropic_example_01"

	w, err := trace.NewWriter(tracePath, sessionID)
	if err != nil {
		log.Fatalf("failed to open trace writer: %v", err)
	}
	defer func() {
		if err := w.Close(); err != nil {
			log.Printf("warning: close writer: %v", err)
		}
	}()

	fmt.Printf("Recording trace to %s ...\n\n", tracePath)

	// --- Simulated tool call 1: read_file (success) ---
	readArgs := mustMarshal(map[string]string{"path": "/etc/hosts"})
	if err := trace.Record(w, "read_file", readArgs, func() (json.RawMessage, error) {
		return mustMarshal(map[string]interface{}{
			"content": "127.0.0.1 localhost\n::1 localhost\n",
			"size":    36,
		}), nil
	}); err != nil {
		log.Fatalf("record read_file: %v", err)
	}
	fmt.Println("[recorded] read_file — success")

	// --- Simulated tool call 2: run_bash (success) ---
	bashArgs := mustMarshal(map[string]string{"command": "echo hello"})
	if err := trace.Record(w, "run_bash", bashArgs, func() (json.RawMessage, error) {
		return mustMarshal(map[string]interface{}{
			"stdout":    "hello\n",
			"exit_code": 0,
		}), nil
	}); err != nil {
		log.Fatalf("record run_bash: %v", err)
	}
	fmt.Println("[recorded] run_bash — success")

	// --- Simulated tool call 3: write_file (failure) ---
	writeArgs := mustMarshal(map[string]interface{}{
		"path":    "/etc/shadow",
		"content": "malicious",
	})
	if err := trace.Record(w, "write_file", writeArgs, func() (json.RawMessage, error) {
		return nil, errors.New("permission denied: /etc/shadow")
	}); err != nil {
		log.Fatalf("record write_file: %v", err)
	}
	fmt.Println("[recorded] write_file — failure (expected)")

	fmt.Printf("\nDone. Trace written to %s\n", tracePath)
	fmt.Println("Replay with:")
	fmt.Printf("  tooltrace replay --mock examples/anthropic/mocks.json %s\n", tracePath)

	writeMocksFile()
}

func writeMocksFile() {
	mocks := map[string]interface{}{
		"read_file": map[string]interface{}{
			"result": map[string]interface{}{
				"content": "127.0.0.1 localhost\n",
				"size":    36,
			},
		},
		"run_bash": map[string]interface{}{
			"result": map[string]interface{}{
				"stdout": "hello\n", "exit_code": 0,
			},
		},
		"write_file": map[string]interface{}{
			"error": "permission denied: /etc/shadow",
		},
	}

	data, err := json.MarshalIndent(mocks, "", "  ")
	if err != nil {
		log.Fatalf("marshal mocks: %v", err)
	}
	if err := os.WriteFile("mocks.json", data, 0o644); err != nil {
		log.Fatalf("write mocks.json: %v", err)
	}
	fmt.Println("Mock file written to mocks.json")
}

func mustMarshal(v interface{}) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("mustMarshal: %v", err))
	}
	return b
}
