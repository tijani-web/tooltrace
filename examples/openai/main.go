// Package main demonstrates tooltrace library usage with an OpenAI-compatible
// tool-calling loop. No live LLM connection is required — tool call boundaries
// and responses are simulated inline to show exactly how Record() is used in a
// real agent loop.
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
	// Write trace to a file in the current directory.
	tracePath := "openai_example.jsonl"
	sessionID := "sess_openai_example_01"

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

	// --- Simulated tool call 1: get_weather (success) ---
	weatherArgs := mustMarshal(map[string]string{"city": "Lagos"})
	if err := trace.Record(w, "get_weather", weatherArgs, func() (json.RawMessage, error) {
		// Simulate a successful weather API response.
		return mustMarshal(map[string]interface{}{
			"city":        "Lagos",
			"temperature": 29,
			"unit":        "celsius",
			"condition":   "partly cloudy",
		}), nil
	}); err != nil {
		log.Fatalf("record get_weather: %v", err)
	}
	fmt.Println("[recorded] get_weather — success")

	// --- Simulated tool call 2: web_search (success) ---
	searchArgs := mustMarshal(map[string]string{"query": "golang tooltrace library"})
	if err := trace.Record(w, "web_search", searchArgs, func() (json.RawMessage, error) {
		return mustMarshal(map[string]interface{}{
			"results": []map[string]string{
				{"title": "tooltrace GitHub", "url": "https://github.com/ToolTraceHQ/tooltrace"},
			},
		}), nil
	}); err != nil {
		log.Fatalf("record web_search: %v", err)
	}
	fmt.Println("[recorded] web_search — success")

	// --- Simulated tool call 3: send_notification (failure) ---
	notifyArgs := mustMarshal(map[string]string{"message": "Build complete", "channel": "slack"})
	if err := trace.Record(w, "send_notification", notifyArgs, func() (json.RawMessage, error) {
		// Simulate a downstream service failure.
		return nil, errors.New("notification service unavailable: connection refused")
	}); err != nil {
		log.Fatalf("record send_notification: %v", err)
	}
	fmt.Println("[recorded] send_notification — failure (expected)")

	fmt.Printf("\nDone. Trace written to %s\n", tracePath)
	fmt.Println("Replay with:")
	fmt.Printf("  tooltrace replay --mock examples/openai/mocks.json %s\n", tracePath)

	// Write a matching mocks.json alongside this example for easy replay.
	writeMocksFile()
}

func writeMocksFile() {
	mocks := map[string]interface{}{
		"get_weather": map[string]interface{}{
			"result": map[string]interface{}{
				"city": "Lagos", "temperature": 29, "unit": "celsius",
			},
		},
		"web_search": map[string]interface{}{
			"result": map[string]interface{}{
				"results": []interface{}{},
			},
		},
		"send_notification": map[string]interface{}{
			"error": "notification service unavailable: connection refused",
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
