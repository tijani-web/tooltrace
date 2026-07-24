package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ToolTraceHQ/tooltrace/pkg/trace"
)

// This example simulates a complex, multi-step OpenAI function calling loop.
// It generates an in-depth trace file (openai_example.jsonl) and a corresponding mocks file.
func main() {
	tracePath := "openai_example.jsonl"
	sessionID := "sess_openai_prod_994"

	// Remove previous trace if it exists so we start fresh
	os.Remove(tracePath)

	w, err := trace.NewWriter(tracePath, sessionID)
	if err != nil {
		log.Fatalf("Failed to open trace writer: %v", err)
	}
	defer w.Close()

	fmt.Printf("Recording deep trace to %s ...\n\n", tracePath)

	// Step 1: User lookup (success)
	args1 := json.RawMessage(`{"email": "admin@example.com", "include_metadata": true}`)
	err = trace.Record(w, "lookup_user", args1, func() (json.RawMessage, error) {
		time.Sleep(45 * time.Millisecond) // simulate latency
		return json.RawMessage(`{
			"user_id": "usr_99812",
			"status": "active",
			"roles": ["admin", "billing_manager"],
			"metadata": {"last_login": "2026-07-24T10:00:00Z", "mfa_enabled": true}
		}`), nil
	})
	printStatus("lookup_user", err)

	// Step 2: Database query (success, complex nested data)
	args2 := json.RawMessage(`{"query": "SELECT * FROM billing_events WHERE user_id = 'usr_99812' ORDER BY created_at DESC LIMIT 2"}`)
	err = trace.Record(w, "execute_sql_query", args2, func() (json.RawMessage, error) {
		time.Sleep(120 * time.Millisecond)
		return json.RawMessage(`{
			"rows_returned": 2,
			"columns": ["id", "amount", "status"],
			"data": [
				{"id": "evt_1", "amount": 49.99, "status": "paid"},
				{"id": "evt_2", "amount": 12.00, "status": "refunded"}
			]
		}`), nil
	})
	printStatus("execute_sql_query", err)

	// Step 3: Trigger external API (failure - rate limit)
	args3 := json.RawMessage(`{"action": "issue_refund", "event_id": "evt_1", "reason": "customer_request"}`)
	err = trace.Record(w, "stripe_api_call", args3, func() (json.RawMessage, error) {
		time.Sleep(80 * time.Millisecond)
		return nil, fmt.Errorf("HTTP 429 Too Many Requests: Stripe API rate limit exceeded")
	})
	printStatus("stripe_api_call", err)

	// Step 4: Fallback notification (success)
	args4 := json.RawMessage(`{"channel": "#billing-alerts", "severity": "high", "message": "Failed to refund evt_1 due to rate limit."}`)
	err = trace.Record(w, "send_slack_alert", args4, func() (json.RawMessage, error) {
		time.Sleep(25 * time.Millisecond)
		return json.RawMessage(`{"delivered": true, "timestamp": "1690000000"}`), nil
	})
	printStatus("send_slack_alert", err)

	fmt.Printf("\nDone. Trace written to %s\n", tracePath)
	
	// Write a matching mocks.json for this complex trace
	writeMocksFile()
}

func printStatus(tool string, err error) {
	if err != nil {
		fmt.Printf("[recorded] %s — failure (expected)\n", tool)
	} else {
		fmt.Printf("[recorded] %s — success\n", tool)
	}
}

func writeMocksFile() {
	data := []byte(`{
  "lookup_user": {
    "result": {
      "user_id": "usr_99812",
      "status": "active",
      "roles": ["admin", "billing_manager"],
      "metadata": {"last_login": "2026-07-24T10:00:00Z", "mfa_enabled": true}
    }
  },
  "execute_sql_query": {
    "result": {
      "rows_returned": 2,
      "columns": ["id", "amount", "status"],
      "data": [
        {"id": "evt_1", "amount": 49.99, "status": "paid"},
        {"id": "evt_2", "amount": 12.00, "status": "refunded"}
      ]
    }
  },
  "stripe_api_call": {
    "error": "HTTP 429 Too Many Requests: Stripe API rate limit exceeded"
  },
  "send_slack_alert": {
    "result": {"delivered": true, "timestamp": "1690000000"}
  }
}`)
	if err := os.WriteFile("mocks.json", data, 0o644); err != nil {
		fmt.Printf("failed to write mocks file: %v\n", err)
	}
}
