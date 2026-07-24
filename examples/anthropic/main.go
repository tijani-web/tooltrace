package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ToolTraceHQ/tooltrace/pkg/trace"
)

// This example simulates an Anthropic Claude Computer Use / Bash execution loop.
// It generates an in-depth trace file (anthropic_example.jsonl) and a corresponding mocks file.
func main() {
	tracePath := "anthropic_example.jsonl"
	sessionID := "sess_claude_sysadmin_02"

	os.Remove(tracePath)

	w, err := trace.NewWriter(tracePath, sessionID)
	if err != nil {
		log.Fatalf("Failed to open trace writer: %v", err)
	}
	defer w.Close()

	fmt.Printf("Recording deep trace to %s ...\n\n", tracePath)

	// Step 1: Read system logs
	args1 := json.RawMessage(`{"command": "cat /var/log/syslog | tail -n 5"}`)
	err = trace.Record(w, "bash_execute", args1, func() (json.RawMessage, error) {
		time.Sleep(60 * time.Millisecond)
		return json.RawMessage(`{
			"stdout": "Jul 24 10:12:00 server sshd[123]: Accepted publickey for root\nJul 24 10:12:01 server systemd[1]: Started Session 4 of user root.",
			"stderr": "",
			"exit_code": 0
		}`), nil
	})
	printStatus("bash_execute", err)

	// Step 2: Attempt to restart a failing service (failure)
	args2 := json.RawMessage(`{"command": "systemctl restart postgresql"}`)
	err = trace.Record(w, "bash_execute", args2, func() (json.RawMessage, error) {
		time.Sleep(300 * time.Millisecond)
		return nil, fmt.Errorf("Job for postgresql.service failed because the control process exited with error code.")
	})
	printStatus("bash_execute", err)

	// Step 3: Check disk space
	args3 := json.RawMessage(`{"command": "df -h /var/lib/postgresql"}`)
	err = trace.Record(w, "bash_execute", args3, func() (json.RawMessage, error) {
		time.Sleep(40 * time.Millisecond)
		return json.RawMessage(`{
			"stdout": "Filesystem      Size  Used Avail Use% Mounted on\n/dev/sda1        50G   50G     0 100% /",
			"stderr": "",
			"exit_code": 0
		}`), nil
	})
	printStatus("bash_execute", err)

	fmt.Printf("\nDone. Trace written to %s\n", tracePath)
	
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
  "bash_execute": {
    "result": {
      "stdout": "Filesystem      Size  Used Avail Use% Mounted on\n/dev/sda1        50G   50G     0 100% /",
      "stderr": "",
      "exit_code": 0
    }
  }
}`)
	// We'll write to a different mock file for Anthropic so they don't overwrite each other if run sequentially in root
	if err := os.WriteFile("anthropic_mocks.json", data, 0o644); err != nil {
		fmt.Printf("failed to write mocks file: %v\n", err)
	}
}
