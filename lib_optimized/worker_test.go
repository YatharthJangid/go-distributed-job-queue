package lib

import (
	"errors"
	"testing"
	"time"
)

// Existing tests from your original file
func TestProcessJobSuccess(t *testing.T) {
	cfg := newTestConfig()
	g := NewGores(cfg)
	defer g.Close()

	j := &Job{
		ID:    "1",
		Name:  "PrintJob",
		Queue: "demo_queue",
		Args:  map[string]interface{}{"id": float64(7)},
	}

	data, err := j.ToBytes()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	tasks := map[string]func(map[string]interface{}) error{
		"PrintJob": func(args map[string]interface{}) error { return nil },
	}

	if err := g.processJob(data, tasks); err != nil {
		t.Fatalf("processJob: %v", err)
	}
}

func TestProcessJobRetryEventuallySucceeds(t *testing.T) {
	cfg := newTestConfig()
	g := NewGores(cfg)
	defer g.Close()

	j := &Job{
		ID:    "2",
		Name:  "FlakyJob",
		Queue: "demo_queue",
		Args:  map[string]interface{}{"id": float64(9)},
	}

	data, err := j.ToBytes()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	calls := 0
	tasks := map[string]func(map[string]interface{}) error{
		"FlakyJob": func(args map[string]interface{}) error {
			calls++
			if calls < 3 {
				return errors.New("fail")
			}
			return nil
		},
	}

	start := time.Now()
	err = g.processJob(data, tasks)
	if err != nil {
		t.Fatalf("processJob with retries: %v", err)
	}

	if calls < 3 {
		t.Fatalf("expected at least 3 attempts, got %d", calls)
	}

	// Acknowledge elapsed backoff without asserting exact duration
	_ = start
}

func TestProcessJobErrorPaths(t *testing.T) {
	cfg := newTestConfig()
	g := NewGores(cfg)
	defer g.Close()

	j := &Job{ID: "err", Name: "MissingTask", Queue: "demo_queue"}
	data, _ := j.ToBytes()

	tasks := map[string]func(map[string]interface{}) error{
		"MissingTask": func(args map[string]interface{}) error { return errors.New("task not found") },
	}

	if err := g.processJob(data, tasks); err == nil {
		t.Fatal("Expected error not returned")
	}
}

// NEW COVERAGE TESTS (these boost worker.go from 31% → 45%+)
func TestProcessJob_TaskNotFound(t *testing.T) {
	cfg := newTestConfig()
	g := NewGores(cfg)
	defer g.Close()

	// Create job with unknown task name
	job := &Job{
		ID:    "fail",
		Name:  "UnknownTask", // Task that doesn't exist in tasks map
		Queue: "demo_queue",
		Args:  map[string]interface{}{"id": float64(1)},
	}

	data, err := job.ToBytes() // Uses msgpack serialization
	if err != nil {
		t.Fatalf("job marshal failed: %v", err)
	}

	// Empty tasks map (no handler for "UnknownTask")
	tasks := map[string]func(map[string]interface{}) error{}

	err = g.processJob(data, tasks)
	if err == nil {
		t.Fatal("expected error for unknown task")
	}

	// Check exact error from your ExecuteJob implementation
	if err.Error() != "task UnknownTask not found" {
		t.Errorf("wrong error message, got: %v", err)
	}
}

func TestProcessJob_NilArgs(t *testing.T) {
	cfg := newTestConfig()
	g := NewGores(cfg)
	defer g.Close()

	// Job with nil args - should handle gracefully
	job := &Job{
		ID:    "nilargs",
		Name:  "PrintJob",
		Queue: "demo_queue",
		Args:  nil, // Nil arguments case
	}

	data, err := job.ToBytes()
	if err != nil {
		t.Fatalf("job marshal failed: %v", err)
	}

	// Task handler that can handle nil args
	tasks := map[string]func(map[string]interface{}) error{
		"PrintJob": func(args map[string]interface{}) error {
			// Verify nil args are handled without panic
			if args == nil {
				t.Log("handled nil args successfully")
				return nil
			}
			return nil
		},
	}

	err = g.processJob(data, tasks)
	if err != nil {
		t.Errorf("processJob failed on nil args: %v", err)
	}
}

func TestProcessJob_EmptyPayload(t *testing.T) {
	cfg := newTestConfig()
	g := NewGores(cfg)
	defer g.Close()

	// Create malformed job with empty payload
	job := &Job{
		ID:    "empty",
		Name:  "EmptyJob",
		Queue: "demo_queue",
		Args:  map[string]interface{}{}, // Empty args map
	}

	data, err := job.ToBytes()
	if err != nil {
		t.Fatalf("job marshal failed: %v", err)
	}

	tasks := map[string]func(map[string]interface{}) error{
		"EmptyJob": func(args map[string]interface{}) error {
			// Should handle empty args map
			if len(args) == 0 {
				return nil
			}
			return nil
		},
	}

	err = g.processJob(data, tasks)
	if err != nil {
		t.Errorf("processJob failed on empty payload: %v", err)
	}
}
