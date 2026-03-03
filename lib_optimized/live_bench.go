package lib

import (
	"fmt"
	"log"
	"time"
)

// RunLiveBenchmark executes a live, end-to-end throughput test against Redis.
// It is called by main.go when the -bench flag is used.
func RunLiveBenchmark() {
	fmt.Println("🚀 Starting Live Throughput Benchmark...")

	// Load config to get Redis connection details
	// We assume config.json is in the root where the binary runs
	config, err := InitConfig("config.json")
	if err != nil {
		log.Printf("Benchmark: Failed to load config.json: %v", err)
		return
	}

	g := NewGores(config)
	defer g.Close()

	count := 10000
	batchSize := 100
	fmt.Printf("   Enqueuing %d jobs to 'live_benchmark_queue'...\n", count)

	// Prepare a batch of dummy jobs
	batch := make([]map[string]interface{}, batchSize)
	for i := 0; i < batchSize; i++ {
		batch[i] = map[string]interface{}{
			"Name":  "BenchJob",
			"Queue": "live_benchmark_queue",
			"Args":  map[string]interface{}{"id": i},
			"Retry": false,
		}
	}

	start := time.Now()
	for i := 0; i < count; i += batchSize {
		if err := g.EnqueueBatch(batch); err != nil {
			log.Printf("Enqueue error: %v", err)
			return
		}
	}
	elapsed := time.Since(start)

	fmt.Printf("✅ Completed in %s\n", elapsed)
	fmt.Printf("📊 Throughput: %.2f jobs/sec\n", float64(count)/elapsed.Seconds())
}
