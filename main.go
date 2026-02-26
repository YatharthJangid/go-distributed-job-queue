package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	lib "myproject/gores/lib_optimized"
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

func main() {
	configPath := flag.String("c", "config.json", "config")
	mode := flag.String("o", "produce", "produce/consume")
	numWorkers := flag.Int("w", 3, "workers")
	bench := flag.Bool("bench", false, "run benchmarks only") // ADD THIS
	flag.Parse()

	if *bench { // ADD THIS BLOCK
		lib.RunBenchmarks()
		return
	}
	configPath = flag.String("c", "config.json", "config")
	mode = flag.String("o", "produce", "produce/consume")
	numWorkers = flag.Int("w", 3, "workers")
	flag.Parse()

	config, err := lib.InitConfig(*configPath)
	if err != nil {
		log.Fatalf("Config: %v", err)
	}

	g := lib.NewGores(config)
	defer g.Close()

	if *mode == "produce" {
		fmt.Println("🚀 Produce: Batch enqueue...")
		batch := make([]map[string]interface{}, 100)
		for i := 0; i < 100; i++ {
			batch[i] = map[string]interface{}{
				"Name":  "PrintJob",
				"Queue": "demo_queue",
				"Args":  map[string]interface{}{"id": float64(i)},
				"Retry": true,
			}
		}
		start := time.Now()
		if err := g.EnqueueBatch(batch); err != nil {
			log.Fatalf("Enqueue: %v", err)
		}
		fmt.Printf("📤 100 jobs in %v (%.0f jobs/sec)\n", time.Since(start), 100/time.Since(start).Seconds())

		info, _ := g.Info()
		data, _ := json.MarshalIndent(info, "", "  ")
		fmt.Printf("\n📊 Stats:\n%s\n", data)
	} else if *mode == "consume" {
		fmt.Println("🚀 Consume: Starting", *numWorkers, "workers...")
		g.StartWorkers(*numWorkers, tasks)
	} else {
		log.Fatal("Mode: produce or consume")
	}
}
