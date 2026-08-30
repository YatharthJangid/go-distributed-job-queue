package gores

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func TestEnqueue_Idempotency(t *testing.T) {
	cfg := newTestConfig()
	g := NewGores(cfg)
	defer g.Close()

	conn := g.pool.Get()
	_, _ = conn.Do("FLUSHDB")
	conn.Close()

	idempKey := fmt.Sprintf("test-key-%d", time.Now().UnixNano())

	job1 := map[string]interface{}{
		"Name":           "PrintJob",
		"Queue":          "demo_queue",
		"Args":           map[string]interface{}{"order_id": 101.0},
		"IdempotencyKey": idempKey,
		"IdempotencyTTL": 60,
		"Retry":          true,
	}

	// First enqueue should succeed
	if err := g.Enqueue(job1); err != nil {
		t.Fatalf("First Enqueue failed: %v", err)
	}

	// Second enqueue with identical key should be recognized as duplicate
	job2 := map[string]interface{}{
		"Name":           "PrintJob",
		"Queue":          "demo_queue",
		"Args":           map[string]interface{}{"order_id": 101.0},
		"IdempotencyKey": idempKey,
		"IdempotencyTTL": 60,
		"Retry":          true,
	}

	if err := g.Enqueue(job2); err != nil {
		t.Fatalf("Second Enqueue errored: %v", err)
	}

	info, err := g.Info()
	if err != nil {
		t.Fatalf("Info failed: %v", err)
	}

	if info["pending"].(int) != 1 {
		t.Errorf("Expected pending = 1, got %v", info["pending"])
	}
	if info["duplicates"].(int) != 1 {
		t.Errorf("Expected duplicates = 1, got %v", info["duplicates"])
	}

	status, err := g.GetJobStatus(idempKey)
	if err != nil {
		t.Fatalf("GetJobStatus failed: %v", err)
	}
	if status != "enqueued" {
		t.Errorf("Expected status 'enqueued', got '%s'", status)
	}
}

func TestEnqueueBatch_Idempotency(t *testing.T) {
	cfg := newTestConfig()
	g := NewGores(cfg)
	defer g.Close()

	conn := g.pool.Get()
	_, _ = conn.Do("FLUSHDB")
	conn.Close()

	baseKey := fmt.Sprintf("batch-key-%d", time.Now().UnixNano())
	batch := make([]map[string]interface{}, 10)

	// 5 unique keys, followed by 5 duplicate keys of the first 5
	for i := 0; i < 5; i++ {
		batch[i] = map[string]interface{}{
			"Name":           "PrintJob",
			"Queue":          "demo_queue",
			"Args":           map[string]interface{}{"id": float64(i)},
			"IdempotencyKey": fmt.Sprintf("%s-%d", baseKey, i),
			"Retry":          true,
		}
	}
	for i := 5; i < 10; i++ {
		batch[i] = map[string]interface{}{
			"Name":           "PrintJob",
			"Queue":          "demo_queue",
			"Args":           map[string]interface{}{"id": float64(i)},
			"IdempotencyKey": fmt.Sprintf("%s-%d", baseKey, i-5), // Duplicate key!
			"Retry":          true,
		}
	}

	if err := g.EnqueueBatch(batch); err != nil {
		t.Fatalf("EnqueueBatch failed: %v", err)
	}

	info, err := g.Info()
	if err != nil {
		t.Fatalf("Info failed: %v", err)
	}

	if info["pending"].(int) != 5 {
		t.Errorf("Expected pending = 5, got %v", info["pending"])
	}
	if info["duplicates"].(int) != 5 {
		t.Errorf("Expected duplicates = 5, got %v", info["duplicates"])
	}
}

func TestWorker_ExecutionIdempotency(t *testing.T) {
	cfg := newTestConfig()
	g := NewGores(cfg)
	defer g.Close()

	conn := g.pool.Get()
	_, _ = conn.Do("FLUSHDB")
	conn.Close()

	var execCount int32
	tasks := map[string]func(map[string]interface{}) error{
		"PaymentJob": func(args map[string]interface{}) error {
			atomic.AddInt32(&execCount, 1)
			return nil
		},
	}

	idempKey := fmt.Sprintf("exec-key-%d", time.Now().UnixNano())

	job := GetJob()
	job.ID = "job-payment-1"
	job.Name = "PaymentJob"
	job.Queue = "demo_queue"
	job.IdempotencyKey = idempKey
	job.Args["amount"] = 99.99
	data, _ := job.ToBytes()
	PutJob(job)

	// First execution should process the job
	if err := g.processJob(data, tasks); err != nil {
		t.Fatalf("First processJob failed: %v", err)
	}

	if atomic.LoadInt32(&execCount) != 1 {
		t.Errorf("Expected execCount = 1, got %d", execCount)
	}

	status, err := g.GetJobStatus(idempKey)
	if err != nil {
		t.Fatalf("GetJobStatus failed: %v", err)
	}
	if status != "completed" {
		t.Errorf("Expected status 'completed', got '%s'", status)
	}

	// Second execution with identical payload should be deduplicated
	if err := g.processJob(data, tasks); err != nil {
		t.Fatalf("Second processJob failed: %v", err)
	}

	if atomic.LoadInt32(&execCount) != 1 {
		t.Errorf("Expected execCount to remain 1 after duplicate execution, got %d", execCount)
	}
}
