package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
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

const dashboardJobCount = 100

const dashboardHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>GoRes job queue</title>
  <style>
    :root { color-scheme: dark; font-family: system-ui, sans-serif; }
    body { max-width: 900px; margin: 40px auto; padding: 0 20px; background: #10131a; color: #eef2ff; }
    h1 { margin-bottom: 8px; } p { color: #aab3c5; }
    button { padding: 11px 16px; border: 0; border-radius: 8px; background: #6d5dfc; color: white; font-weight: 700; cursor: pointer; }
    button:disabled { opacity: .55; cursor: wait; }
    .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(150px, 1fr)); gap: 12px; margin: 24px 0; }
    .card { padding: 18px; border: 1px solid #2a3140; border-radius: 10px; background: #171c26; }
    .value { display: block; margin-top: 6px; font-size: 1.7rem; font-weight: 700; }
    #status { min-height: 1.5em; margin-top: 18px; color: #9fe6b8; }
  </style>
</head>
<body>
  <h1>GoRes distributed job queue</h1>
  <p>Redis-backed jobs, pooled objects, retries, idempotency, and concurrent workers.</p>
  <button id="run" onclick="runDemo()">Run 100-job demo</button>
  <div id="status">Loading queue statistics…</div>
  <section class="grid">
    <div class="card">Processed<span class="value" id="processed">—</span></div>
    <div class="card">Pending<span class="value" id="pending">—</span></div>
    <div class="card">Enqueued<span class="value" id="enqueued">—</span></div>
    <div class="card">Duplicates<span class="value" id="duplicates">—</span></div>
    <div class="card">Workers<span class="value" id="workers">—</span></div>
  </section>
  <script>
    let run = null;
    const $ = id => document.getElementById(id);

    async function refresh() {
      try {
        const response = await fetch('/api/info', {cache: 'no-store'});
        if (!response.ok) throw new Error('stats unavailable');
        const stats = await response.json();
        ['processed', 'pending', 'enqueued', 'duplicates'].forEach(key => $(key).textContent = stats[key]);
        $('workers').textContent = stats.workers + ' local';
        if (run && stats.processed >= run.target && stats.pending === 0) {
          const seconds = Math.max(0.001, stats.demo_elapsed_ms / 1000);
          $('status').textContent = run.count + ' jobs finished in ' + seconds.toFixed(2) + ' seconds (' + Math.round(run.count / seconds) + ' jobs/sec)';
          $('run').disabled = false;
          run = null;
        }
      } catch (error) {
        $('status').textContent = 'Waiting for Redis…';
      }
    }

    async function runDemo() {
      $('run').disabled = true;
      $('status').textContent = 'Enqueuing 100 jobs…';
      try {
        const response = await fetch('/api/demo', {method: 'POST'});
        if (!response.ok) throw new Error(await response.text());
        const demo = await response.json();
        run = {count: demo.count, target: demo.target};
        $('status').textContent = 'Workers are processing the jobs…';
      } catch (error) {
        $('status').textContent = error.message || 'Demo failed';
        $('run').disabled = false;
      }
    }

    refresh();
    setInterval(refresh, 1000);
  </script>
</body>
</html>`

func makeDemoBatch(count int, useIdemp bool) []map[string]interface{} {
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
	return batch
}

func runProducer(g *gores.Gores, count int, useIdemp bool) {
	fmt.Printf("🚀 Produce: Batch enqueue (%d jobs, idempotency: %v)...\n", count, useIdemp)
	batch := makeDemoBatch(count, useIdemp)
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
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	var demoMu sync.Mutex
	demoRunning, demoTarget := false, 0
	var demoElapsed time.Duration
	var demoStartedAt time.Time
	mux := http.NewServeMux()
	writeJSON := func(w http.ResponseWriter, value interface{}) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(value)
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprint(w, dashboardHTML)
	})
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/api/info", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		info, err := g.Info()
		if err != nil {
			http.Error(w, "redis unavailable", http.StatusServiceUnavailable)
			return
		}
		processed, _ := info["processed"].(int)
		pending, _ := info["pending"].(int)
		demoMu.Lock()
		if demoRunning && processed >= demoTarget && pending == 0 {
			demoElapsed = time.Since(demoStartedAt)
			demoRunning = false
		}
		info["workers"] = numWorkers
		info["demo_running"] = demoRunning
		info["demo_target"] = demoTarget
		info["demo_elapsed_ms"] = demoElapsed.Milliseconds()
		demoMu.Unlock()
		writeJSON(w, info)
	})
	mux.HandleFunc("/api/demo", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		demoMu.Lock()
		defer demoMu.Unlock()
		if demoRunning {
			http.Error(w, "demo already running", http.StatusConflict)
			return
		}
		info, err := g.Info()
		if err != nil {
			http.Error(w, "redis unavailable", http.StatusServiceUnavailable)
			return
		}
		processed, _ := info["processed"].(int)
		batch := makeDemoBatch(dashboardJobCount, false)
		if err := g.EnqueueBatch(batch); err != nil {
			http.Error(w, "enqueue failed", http.StatusServiceUnavailable)
			return
		}
		demoTarget = processed + len(batch)
		demoStartedAt = time.Now()
		demoElapsed = 0
		demoRunning = true
		writeJSON(w, map[string]interface{}{
			"count":  len(batch),
			"target": demoTarget,
		})
	})

	go func() {
		if err := http.ListenAndServe(":"+port, mux); err != nil {
			log.Printf("Dashboard server: %v", err)
		}
	}()
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
