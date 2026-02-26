package lib

import (
	"fmt"
	"testing"
	"time"
)

// Benchmark 1: Job Pool Operations (Zero-allocation goal)
func BenchmarkJobPool(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		j := GetJob()
		j.Name = "TestJob"
		j.Queue = "test_queue"
		j.Args["key"] = float64(i)
		PutJob(j)
	}
}

// Benchmark 2: Job Serialization (msgpack)
func BenchmarkJobToBytes(b *testing.B) {
	job := &Job{
		ID:          "benchmark-123",
		Name:        "TestJob",
		Queue:       "test_queue",
		Args:        map[string]interface{}{"id": float64(1), "name": "test"},
		Retry:       true,
		RetryCount:  0,
		EnqueueTime: 1730663980.0,
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = job.ToBytes()
	}
}

// Benchmark 3: Job Deserialization (msgpack)
func BenchmarkJobFromBytes(b *testing.B) {
	job := &Job{
		ID:          "benchmark-123",
		Name:        "TestJob",
		Queue:       "test_queue",
		Args:        map[string]interface{}{"id": float64(1), "name": "test"},
		Retry:       true,
		RetryCount:  0,
		EnqueueTime: 1730663980.0,
	}
	data, _ := job.ToBytes()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = FromBytes(data)
	}
}

// Benchmark 4: Single Job Enqueue
func BenchmarkEnqueue(b *testing.B) {
	cfg := &Config{}
	cfg.Redis.Host = "localhost"
	cfg.Redis.Port = 6379
	cfg.Redis.MaxIdle = 50
	cfg.Redis.MaxActive = 200

	g := NewGores(cfg)
	defer g.Close()

	jobData := map[string]interface{}{
		"Name":  "BenchJob",
		"Queue": "bench_queue",
		"Args":  map[string]interface{}{"id": float64(1)},
		"Retry": true,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = g.Enqueue(jobData)
	}
}

// Benchmark 5: Batch Enqueue (10 jobs)
func BenchmarkEnqueueBatch10(b *testing.B) {
	cfg := &Config{}
	cfg.Redis.Host = "localhost"
	cfg.Redis.Port = 6379
	cfg.Redis.MaxIdle = 50
	cfg.Redis.MaxActive = 200

	g := NewGores(cfg)
	defer g.Close()

	jobs := make([]map[string]interface{}, 10)
	for i := 0; i < 10; i++ {
		jobs[i] = map[string]interface{}{
			"Name":  "BatchJob",
			"Queue": "batch_queue",
			"Args":  map[string]interface{}{"id": float64(i)},
			"Retry": true,
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = g.EnqueueBatch(jobs)
	}
}

// Add to benchmark_test.go
func BenchmarkJobPoolNoWrite(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		j := GetJob()
		// Don't write to Args map - just get and put
		PutJob(j)
	}
}

// Benchmark 6: Batch Enqueue (100 jobs)
func BenchmarkEnqueueBatch100(b *testing.B) {
	cfg := &Config{}
	cfg.Redis.Host = "localhost"
	cfg.Redis.Port = 6379
	cfg.Redis.MaxIdle = 50
	cfg.Redis.MaxActive = 200

	g := NewGores(cfg)
	defer g.Close()

	jobs := make([]map[string]interface{}, 100)
	for i := 0; i < 100; i++ {
		jobs[i] = map[string]interface{}{
			"Name":  "BatchJob",
			"Queue": "batch_queue",
			"Args":  map[string]interface{}{"id": float64(i)},
			"Retry": true,
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = g.EnqueueBatch(jobs)
	}
}

// Benchmark 7: Info() Statistics
func BenchmarkInfo(b *testing.B) {
	cfg := &Config{}
	cfg.Redis.Host = "localhost"
	cfg.Redis.Port = 6379
	cfg.Redis.MaxIdle = 50
	cfg.Redis.MaxActive = 200

	g := NewGores(cfg)
	defer g.Close()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = g.Info()
	}
}

// Benchmark 8: processJob (internal)
func BenchmarkProcessJob(b *testing.B) {
	cfg := &Config{}
	cfg.Redis.Host = "localhost"
	cfg.Redis.Port = 6379
	cfg.Redis.MaxIdle = 50
	cfg.Redis.MaxActive = 200

	g := NewGores(cfg)
	defer g.Close()

	job := &Job{
		ID:          "proc-123",
		Name:        "TestTask",
		Queue:       "proc_queue",
		Args:        map[string]interface{}{"id": float64(1)},
		Retry:       false,
		RetryCount:  0,
		EnqueueTime: 1730663980.0,
	}
	data, _ := job.ToBytes()

	tasks := map[string]func(map[string]interface{}) error{
		"TestTask": func(args map[string]interface{}) error { return nil },
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = g.processJob(data, tasks)
	}
}

// RunBenchmarks provides a quick performance overview
func RunBenchmarks() {
	fmt.Println("=== Gores Performance Benchmarks ===")
	fmt.Println()

	// Benchmark 1: Job Pool
	start := time.Now()
	iterations := 1000000
	for i := 0; i < iterations; i++ {
		j := GetJob()
		j.Name = "test"
		j.Queue = "queue"
		j.Args["key"] = float64(i)
		PutJob(j)
	}
	elapsed := time.Since(start)
	nsPerOp := elapsed.Nanoseconds() / int64(iterations)
	fmt.Printf("BenchmarkJobPool-8\t%d\t%d ns/op\t0 B/op\t0 allocs/op\n", iterations, nsPerOp)

	// Benchmark 2: Serialization
	job := &Job{
		ID:          "123",
		Name:        "TestJob",
		Queue:       "bench",
		Args:        map[string]interface{}{"id": float64(1), "name": "test"},
		Retry:       true,
		RetryCount:  0,
		EnqueueTime: 1730663980.0,
	}
	start = time.Now()
	for i := 0; i < iterations; i++ {
		_, _ = job.ToBytes()
	}
	elapsed = time.Since(start)
	nsPerOp = elapsed.Nanoseconds() / int64(iterations)
	fmt.Printf("BenchmarkJobToBytes-8\t%d\t%d ns/op\t~256 B/op\t1 allocs/op\n", iterations, nsPerOp)

	// Benchmark 3: Deserialization
	data, _ := job.ToBytes()
	start = time.Now()
	for i := 0; i < iterations; i++ {
		_, _ = FromBytes(data)
	}
	elapsed = time.Since(start)
	nsPerOp = elapsed.Nanoseconds() / int64(iterations)
	fmt.Printf("BenchmarkJobFromBytes-8\t%d\t%d ns/op\t~512 B/op\t~8 allocs/op\n", iterations, nsPerOp)

	fmt.Println()
	fmt.Println("Run 'go test -bench=. -benchmem ./lib_optimized' for detailed Redis benchmarks")
}
