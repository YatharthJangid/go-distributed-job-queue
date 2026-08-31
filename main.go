package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"myproject/gores/pkg/gores"
)

var tasks = map[string]func(map[string]interface{}) error{
	"PrintJob": func(args map[string]interface{}) error {
		id := int(args["id"].(float64))
		fmt.Printf("✅ PrintJob ID: %d at %s\n", id, time.Now().Format("15:04:05"))
		return nil
	},
	"CalcJob": func(args map[string]interface{}) error {
		a := args["a"].(float64)
		b := args["b"].(float64)
		fmt.Printf("🧮 Calc: %.2f * %.2f = %.2f\n", a, b, a*b)
		return nil
	},
}

func runProducer(g *gores.Gores, count int, useIdemp bool) {
	fmt.Printf("🚀 Produce: Batch enqueue (%d jobs, idempotency: %v)...\n", count, useIdemp)
	batch := make([]map[string]interface{}, count)
	for i := 0; i < count; i++ {
		job := map[string]interface{}{
			"Name":  "PrintJob",
			"Queue": "demo_queue",
			"Args":  map[string]interface{}{"id": float64(i)},
			"Retry": true,
		}
		if useIdemp {
			job["IdempotencyKey"] = fmt.Sprintf("job-idemp-%d", i)
			job["IdempotencyTTL"] = 300
		}
		batch[i] = job
	}
	start := time.Now()
	if err := g.EnqueueBatch(batch); err != nil {
		log.Fatalf("Enqueue: %v", err)
	}
	duration := time.Since(start)
	jobsPerSec := float64(count) / duration.Seconds()
	if duration.Seconds() == 0 {
		jobsPerSec = float64(count)
	}
	fmt.Printf("📤 %d jobs in %v (%.0f jobs/sec)\n", count, duration, jobsPerSec)

	info, _ := g.Info()
	data, _ := json.MarshalIndent(info, "", "  ")
	fmt.Printf("\n📊 Stats:\n%s\n", data)
}

func runConsumer(g *gores.Gores, numWorkers int) {
	fmt.Println("🚀 Consume: Starting", numWorkers, "workers...")
	if port := os.Getenv("PORT"); port != "" {
		http.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		go func() {
			if err := http.ListenAndServe(":"+port, nil); err != nil {
				log.Printf("Health server: %v", err)
			}
		}()
	}
	g.StartWorkers(numWorkers, tasks)
}

func main() {
	configPath := flag.String("c", "config.json", "config")
	mode := flag.String("o", "produce", "produce/consume")
	numWorkers := flag.Int("w", 3, "workers")
	jobCount := flag.Int("n", 100, "number of jobs to produce")
	useIdemp := flag.Bool("idemp", false, "enable idempotency keys on produced jobs")
	bench := flag.Bool("bench", false, "run benchmarks only")
	flag.Parse()

	if *bench {
		gores.RunLiveBenchmark()
		return
	}

	config, err := gores.InitConfig(*configPath)
	if err != nil {
		log.Fatalf("Config: %v", err)
	}

	g := gores.NewGores(config)
	defer g.Close()

	switch *mode {
	case "produce":
		runProducer(g, *jobCount, *useIdemp)
	case "consume":
		runConsumer(g, *numWorkers)
	default:
		log.Fatal("Mode must be 'produce' or 'consume'")
	}
}
