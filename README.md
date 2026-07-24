# tooltrace
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Reference](https://pkg.go.dev/badge/github.com/ToolTraceHQ/tooltrace.svg)](https://pkg.go.dev/github.com/ToolTraceHQ/tooltrace/pkg/trace)
[![Go Report Card](https://goreportcard.com/badge/github.com/ToolTraceHQ/tooltrace)](https://goreportcard.com/report/github.com/ToolTraceHQ/tooltrace)

**Record and replay LLM tool calls.**

`tooltrace` sits between your LLM agent and the tools it calls. It records every tool invocation to a portable trace file, and lets you replay, validate, diff, and inspect that trace later — without depending on any specific LLM provider, framework, or SDK.

---

## ⚠️ Security / Privacy Notice

**Trace files record raw tool arguments and results verbatim.** This means they may capture API keys, auth tokens, passwords, or PII that were passed to or returned by your tools. 

In v1 and v2, there is **no automatic redaction**. You are fully responsible for securing your trace files:
- Add `*.jsonl` to your `.gitignore`.
- Restrict file permissions in production.
- Do not attach raw trace files to public GitHub issues or share them on Slack without manual scrubbing.

*Note: Automatic, field-level redaction is on the roadmap for v3.*

---

## Why this exists

Every agent framework logs tool calls differently. Most don't validate arguments before executing. When an agent misbehaves in production, there is usually no clean, portable way to replay the exact sequence of tool calls locally to debug the issue—you have to re-run the whole expensive session and hope it reproduces.

`tooltrace` treats the **tool invocation** as the stable primitive. Whether you are using OpenAI, Anthropic, Gemini, Ollama, or a custom loop, a tool call is just `(name, arguments) -> result`. By recording this to a standard format, you can decouple your debugging, evaluation, and guardrails from your specific LLM provider.

### Core Features (v1)
- **Zero dependencies:** Built entirely on the Go standard library.
- **Provider-agnostic:** Works with any LLM framework.
- **Concurrency-safe:** The Go logger safely handles concurrent tool calls out of the box.
- **Standard format:** Append-only `.jsonl` trace files are portable and grep-able.

---

## Installation

Install the CLI (used for replaying traces):
```bash
go install github.com/ToolTraceHQ/tooltrace/cmd/tooltrace@latest
```

Add the library to your Go project (used for recording traces):
```bash
go get github.com/ToolTraceHQ/tooltrace/pkg/trace
```

---

## Usage

**Recording is done via the Go library. Replaying is done via the CLI.**

### 1. Recording a Trace (Go Library)

Wrap your tool execution in `trace.Record`. It handles the timing, captures successes/failures, and securely appends the invocation to the trace file.

```go
package main

import (
	"encoding/json"
	"github.com/ToolTraceHQ/tooltrace/pkg/trace"
)

func main() {
	// 1. Open a trace writer for a specific session ID
	w, _ := trace.NewWriter("trace.jsonl", "sess_01")
	defer w.Close()

	args := json.RawMessage(`{"city": "Lagos"}`)
	
	// 2. Wrap your tool execution
	err := trace.Record(w, "weather", args, func() (json.RawMessage, error) {
		// Your actual tool execution logic here...
		// On success, return the JSON result and nil.
		// On failure, return nil and the error.
		return json.RawMessage(`{"temperature": 29}`), nil
	})
	
	if err != nil {
		panic("Failed to record trace: " + err.Error())
	}
}
```

### 2. Replaying a Trace (CLI)

You can replay a recorded trace locally to verify that your agent's tool calls are functioning as expected. To do this, you provide a mock registry.

First, create a `mocks.json` file. Each tool name maps to either a `"result"` (simulated success) or an `"error"` (simulated failure).

```json
{
  "weather": { 
    "result": { "temperature": 29 } 
  },
  "send_sms": { 
    "error": "service unavailable: connection refused" 
  }
}
```

Then, run the replay command:

```bash
tooltrace replay --mock mocks.json trace.jsonl
```

**Output:**
```text
[PASS] step 0 — weather
[PASS] step 1 — send_sms

2 passed, 0 failed
```
*A step passes if the recorded outcome (success or failure) matches the outcome provided by your mock registry.*

---

## Examples

Check out the `/examples` directory for complete, runnable demonstrations:
- [`examples/openai/main.go`](examples/openai/main.go) — Demonstrates an OpenAI-compatible tool calling loop.
- [`examples/anthropic/main.go`](examples/anthropic/main.go) — Demonstrates an Anthropic/Claude-style tool use loop.

---

## Specification and Roadmap

The core trace format is an append-only JSONL schema. It is a public contract and is intentionally stable. 

For the complete specification of the trace format and our roadmap for future versions, see [TOOLTRACE.md](TOOLTRACE.md). 

Currently, only **v1** (Record + Replay) is implemented. Advanced features like JSON Schema validation, diffing traces, policy guardrails, and redaction are planned for v2 and v3.

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for details on how to run the test suite, format your code, and submit bug reports.

## License

This project is licensed under the [MIT License](LICENSE).
