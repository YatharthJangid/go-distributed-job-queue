package lib

import (
	"encoding/json"
	"testing"
)

func newTestConfig() *Config {
	cfg := &Config{}
	cfg.Redis.Host = "localhost"
	cfg.Redis.Port = 6379
	cfg.Redis.PoolSize = 10
	cfg.Redis.MaxIdle = 50
	cfg.Redis.MaxActive = 200
	return cfg
}

func TestEnqueueAndInfoCounts(t *testing.T) {
	cfg := newTestConfig()
	g := NewGores(cfg)
	defer g.Close()

	// Clean keys used by Info (demo_queue)
	conn := g.pool.Get()
	_, _ = conn.Do("DEL",
		g.prefix+"demo_queue"+QUEUE_PENDING,
		g.prefix+"demo_queue"+QUEUE_PROCESS,
		g.prefix+STAT_ENQUEUED,
		g.prefix+STAT_PROCESSED,
	)
	conn.Close()

	// Single enqueue -> pending:1
	job := map[string]interface{}{
		"Name":  "PrintJob",
		"Queue": "demo_queue",
		"Args":  map[string]interface{}{"id": float64(1)},
		"Retry": true,
	}
	if err := g.Enqueue(job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	info, err := g.Info()
	if err != nil {
		t.Fatalf("info: %v", err)
	}
	b, _ := json.Marshal(info)
	_ = b // keep linter happy

	if info["pending"].(int) < 1 {
		t.Fatalf("expected pending >=1, got %#v", info["pending"])
	}
}

func TestEnqueueBatch100(t *testing.T) {
	cfg := newTestConfig()
	g := NewGores(cfg)
	defer g.Close()

	conn := g.pool.Get()
	_, _ = conn.Do("DEL", g.prefix+"demo_queue"+QUEUE_PENDING)
	conn.Close()

	batch := make([]map[string]interface{}, 100)
	for i := 0; i < 100; i++ {
		batch[i] = map[string]interface{}{
			"Name":  "PrintJob",
			"Queue": "demo_queue",
			"Args":  map[string]interface{}{"id": float64(i)},
			"Retry": true,
		}
	}
	if err := g.EnqueueBatch(batch); err != nil {
		t.Fatalf("batch: %v", err)
	}

	info, err := g.Info()
	if err != nil {
		t.Fatalf("info: %v", err)
	}
	if info["pending"].(int) < 100 {
		t.Fatalf("expected pending >=100, got %#v", info["pending"])
	}
	if _, ok := info["Enqueue_timestamp"]; !ok {
		t.Fatalf("missing Enqueue_timestamp")
	}
}
