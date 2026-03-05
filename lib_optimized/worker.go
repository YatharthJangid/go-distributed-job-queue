package lib

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"
)

func (g *Gores) StartWorkers(n int, tasks map[string]func(map[string]interface{}) error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		log.Println("Received termination signal, shutting down workers gracefully...")
		cancel()
	}()

	numCPU := runtime.NumCPU()
	var wg sync.WaitGroup

	for i := 0; i < n; i++ {
		wg.Add(1)
		core := i % numCPU
		go func(workerID, coreID int) {
			defer wg.Done()
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()

			conn := g.pool.Get()
			defer conn.Close()

			for {
				select {
				case <-ctx.Done():
					return
				default:
					reply, err := conn.Do("BRPOPLPUSH", g.prefix+"demo_queue"+QUEUE_PENDING, g.prefix+"demo_queue"+QUEUE_PROCESS, 1)
					if err != nil || reply == nil {
						if err != nil {
							conn.Close()
							conn = g.pool.Get()
							time.Sleep(time.Second) // backoff
						}
						continue
					}

					data := reply.([]byte)
					if err := g.processJob(data, tasks); err != nil {
						log.Printf("Worker %d failed job after retries: %v", workerID, err)
						_, _ = conn.Do("LPUSH", g.prefix+"demo_queue_deadletter", data)
					}
					_, _ = conn.Do("LREM", g.prefix+"demo_queue"+QUEUE_PROCESS, 1, data)
				}
			}
		}(i, core)
	}

	wg.Wait()
	log.Println("All workers shut down.")
}

func (g *Gores) processJob(data []byte, tasks map[string]func(map[string]interface{}) error) error {
	job, err := FromBytes(data)
	if err != nil {
		return err
	}
	defer PutJob(job)

	fn, ok := tasks[job.Name]
	if !ok {
		return fmt.Errorf("task %s not found", job.Name)
	}

	for r := 0; r < 3; r++ {
		if err := fn(job.Args); err == nil {
			return nil
		}
		time.Sleep(time.Duration(math.Pow(2, float64(r))) * time.Second)
	}
	return fmt.Errorf("max retries")
}
