package trace_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/ToolTraceHQ/tooltrace/pkg/trace"
)

// --- helpers ---

func tempFile(t *testing.T) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "trace-*.jsonl")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	f.Close()
	return f.Name()
}

func mustJSON(t *testing.T, v interface{}) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	return b
}

// ===== Writer / Record tests =====

func TestRecord_Success(t *testing.T) {
	path := tempFile(t)
	w, err := trace.NewWriter(path, "sess_test_01")
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	defer w.Close()

	args := mustJSON(t, map[string]string{"city": "Lagos"})
	result := mustJSON(t, map[string]int{"temperature": 29})

	err = trace.Record(w, "get_weather", args, func() (json.RawMessage, error) {
		return result, nil
	})
	if err != nil {
		t.Fatalf("Record: %v", err)
	}

	calls, err := trace.Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	c := calls[0]
	if c.Tool != "get_weather" {
		t.Errorf("Tool = %q; want %q", c.Tool, "get_weather")
	}
	if c.Error != nil {
		t.Errorf("Error should be nil on success, got %q", *c.Error)
	}
	if c.Result == nil {
		t.Error("Result should be populated on success")
	}
	if c.SessionID != "sess_test_01" {
		t.Errorf("SessionID = %q; want %q", c.SessionID, "sess_test_01")
	}
	if c.Sequence != 0 {
		t.Errorf("Sequence = %d; want 0", c.Sequence)
	}
	if c.DurationMS < 0 {
		t.Errorf("DurationMS = %d; must be >= 0", c.DurationMS)
	}
}

func TestRecord_Failure(t *testing.T) {
	path := tempFile(t)
	w, err := trace.NewWriter(path, "sess_test_02")
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	defer w.Close()

	args := mustJSON(t, map[string]string{"path": "/etc/shadow"})
	recordErr := errors.New("permission denied")

	err = trace.Record(w, "read_file", args, func() (json.RawMessage, error) {
		return nil, recordErr
	})
	if err != nil {
		t.Fatalf("Record: %v", err)
	}

	calls, err := trace.Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	c := calls[0]
	if c.Error == nil {
		t.Fatal("Error should be populated on failure")
	}
	if *c.Error != "permission denied" {
		t.Errorf("Error = %q; want %q", *c.Error, "permission denied")
	}
	if c.Result != nil && string(c.Result) != "null" {
		t.Errorf("Result should be nil or null on failure, got %s", c.Result)
	}
}

func TestRecord_SequenceIncrements(t *testing.T) {
	path := tempFile(t)
	w, err := trace.NewWriter(path, "sess_seq")
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	defer w.Close()

	for i := 0; i < 5; i++ {
		if err := trace.Record(w, "tool", mustJSON(t, nil), func() (json.RawMessage, error) {
			return mustJSON(t, "ok"), nil
		}); err != nil {
			t.Fatalf("Record %d: %v", i, err)
		}
	}

	calls, err := trace.Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(calls) != 5 {
		t.Fatalf("expected 5 calls, got %d", len(calls))
	}
	for i, c := range calls {
		if c.Sequence != int64(i) {
			t.Errorf("calls[%d].Sequence = %d; want %d", i, c.Sequence, i)
		}
	}
}

func TestRecord_ConcurrentWrites(t *testing.T) {
	path := tempFile(t)
	w, err := trace.NewWriter(path, "sess_concurrent")
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	defer w.Close()

	const goroutines = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(n int) {
			defer wg.Done()
			args := mustJSON(t, map[string]int{"n": n})
			if err := trace.Record(w, fmt.Sprintf("tool_%d", n), args, func() (json.RawMessage, error) {
				return mustJSON(t, "ok"), nil
			}); err != nil {
				t.Errorf("goroutine %d Record: %v", n, err)
			}
		}(i)
	}
	wg.Wait()

	calls, err := trace.Read(path)
	if err != nil {
		t.Fatalf("Read after concurrent writes: %v", err)
	}
	if len(calls) != goroutines {
		t.Errorf("expected %d calls after concurrent writes, got %d", goroutines, len(calls))
	}
}

// ===== Reader tests =====

func TestRead_EmptyFile(t *testing.T) {
	path := tempFile(t)
	calls, err := trace.Read(path)
	if err != nil {
		t.Fatalf("Read empty file: %v", err)
	}
	if len(calls) != 0 {
		t.Errorf("expected 0 calls from empty file, got %d", len(calls))
	}
}

func TestRead_SingleLine(t *testing.T) {
	path := tempFile(t)
	w, _ := trace.NewWriter(path, "sess_single")
	trace.Record(w, "tool", mustJSON(t, nil), func() (json.RawMessage, error) {
		return mustJSON(t, "ok"), nil
	})
	w.Close()

	calls, err := trace.Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(calls) != 1 {
		t.Errorf("expected 1 call, got %d", len(calls))
	}
}

func TestRead_ManyLines(t *testing.T) {
	path := tempFile(t)
	w, _ := trace.NewWriter(path, "sess_many")
	for i := 0; i < 100; i++ {
		trace.Record(w, "tool", mustJSON(t, nil), func() (json.RawMessage, error) {
			return mustJSON(t, "ok"), nil
		})
	}
	w.Close()

	calls, err := trace.Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(calls) != 100 {
		t.Errorf("expected 100 calls, got %d", len(calls))
	}
}

func TestRead_MalformedJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.jsonl")
	if err := os.WriteFile(path, []byte("{not valid json}\n"), 0o644); err != nil {
		t.Fatalf("write bad file: %v", err)
	}

	_, err := trace.Read(path)
	if err == nil {
		t.Fatal("expected error from malformed JSON, got nil")
	}
}

func TestRead_FileNotExist(t *testing.T) {
	_, err := trace.Read("/nonexistent/path/trace.jsonl")
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
}

// ===== Replay tests =====

func TestReplay_AllPass(t *testing.T) {
	path := tempFile(t)
	w, _ := trace.NewWriter(path, "sess_replay_pass")
	trace.Record(w, "weather", mustJSON(t, nil), func() (json.RawMessage, error) {
		return mustJSON(t, "sunny"), nil
	})
	trace.Record(w, "search", mustJSON(t, nil), func() (json.RawMessage, error) {
		return mustJSON(t, "results"), nil
	})
	w.Close()

	calls, _ := trace.Read(path)
	registry := trace.MockRegistry{
		"weather": func(args json.RawMessage) (json.RawMessage, error) {
			return mustJSON(t, "sunny"), nil
		},
		"search": func(args json.RawMessage) (json.RawMessage, error) {
			return mustJSON(t, "results"), nil
		},
	}

	results := trace.Replay(calls, registry)
	for _, r := range results {
		if !r.Passed {
			t.Errorf("step %d (%s) should pass but got: %s", r.Sequence, r.Tool, r.Reason)
		}
	}
}

func TestReplay_UnexpectedError(t *testing.T) {
	path := tempFile(t)
	w, _ := trace.NewWriter(path, "sess_replay_unexpected_err")
	// Record a SUCCESS.
	trace.Record(w, "weather", mustJSON(t, nil), func() (json.RawMessage, error) {
		return mustJSON(t, "sunny"), nil
	})
	w.Close()

	calls, _ := trace.Read(path)
	registry := trace.MockRegistry{
		// Mock now returns an error — mismatch.
		"weather": func(args json.RawMessage) (json.RawMessage, error) {
			return nil, errors.New("service down")
		},
	}

	results := trace.Replay(calls, registry)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Passed {
		t.Error("step should fail when mock errors but record succeeded")
	}
}

func TestReplay_MissingMock(t *testing.T) {
	path := tempFile(t)
	w, _ := trace.NewWriter(path, "sess_replay_missing")
	trace.Record(w, "unknown_tool", mustJSON(t, nil), func() (json.RawMessage, error) {
		return mustJSON(t, "ok"), nil
	})
	w.Close()

	calls, _ := trace.Read(path)
	registry := trace.MockRegistry{} // empty — no mock for "unknown_tool"

	results := trace.Replay(calls, registry)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Passed {
		t.Error("step should fail when tool has no mock registered")
	}
}

// ===== Integration test =====

// TestIntegration_RecordThenReplay records a sequence of calls to a temp file
// then replays that same file and asserts all steps pass.
func TestIntegration_RecordThenReplay(t *testing.T) {
	path := tempFile(t)
	w, err := trace.NewWriter(path, "sess_integration")
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}

	tools := []struct {
		name string
		args interface{}
		res  interface{}
	}{
		{"get_weather", map[string]string{"city": "Lagos"}, map[string]int{"temperature": 29}},
		{"web_search", map[string]string{"query": "go"}, map[string]interface{}{"results": []string{"golang.org"}}},
		{"send_email", map[string]string{"to": "a@b.com"}, map[string]bool{"sent": true}},
	}

	registry := make(trace.MockRegistry)
	for _, tool := range tools {
		tool := tool // capture
		args := mustJSON(t, tool.args)
		result := mustJSON(t, tool.res)
		if err := trace.Record(w, tool.name, args, func() (json.RawMessage, error) {
			return result, nil
		}); err != nil {
			t.Fatalf("Record %s: %v", tool.name, err)
		}
		registry[tool.name] = func(a json.RawMessage) (json.RawMessage, error) {
			return mustJSON(t, tool.res), nil
		}
	}
	w.Close()

	calls, err := trace.Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(calls) != len(tools) {
		t.Fatalf("expected %d calls, got %d", len(tools), len(calls))
	}

	results := trace.Replay(calls, registry)
	for _, r := range results {
		if !r.Passed {
			t.Errorf("integration step %d (%s) failed: %s", r.Sequence, r.Tool, r.Reason)
		}
	}
}
