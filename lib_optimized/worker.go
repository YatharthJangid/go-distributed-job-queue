package lib

import (
	"context"
	"fmt"
	"log"
	"math"
	"runtime"
	"time"
)

func (g *Gores) StartWorkers(n int, tasks map[string]func(map[string]interface{}) error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	numCPU := runtime.NumCPU()
	for i := 0; i < n; i++ {
		core := i % numCPU
		go func(workerID, coreID int) {
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()

			for {
				select {
				case <-ctx.Done():
					return
				default:
					conn := g.pool.Get()
					reply, err := conn.Do("BRPOPLPUSH", g.prefix+"demo_queue"+QUEUE_PENDING, g.prefix+"demo_queue"+QUEUE_PROCESS, 1)
					conn.Close()
					if err != nil || reply == nil {
						continue
					}
					data := reply.([]byte)
					if err := g.processJob(data, tasks); err != nil {
						log.Printf("Worker %d: %v", workerID, err)
					}
				}
			}
		}(i, core)
	}
	select {}
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
