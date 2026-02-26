package lib

import (
	"fmt"
	"time"

	"github.com/garyburd/redigo/redis"
)

type Gores struct {
	pool   *redis.Pool
	prefix string
}

const luaEnqueue = `
	local queue = KEYS[1]
	local data = ARGV[1]
	local statKey = KEYS[2]
	redis.call('LPUSH', queue, data)
	redis.call('INCR', statKey)
	return 1
`

func NewGores(config *Config) *Gores {
	pool := &redis.Pool{
		MaxIdle:     config.Redis.MaxIdle,
		MaxActive:   config.Redis.MaxActive,
		IdleTimeout: time.Duration(config.Redis.IdleTimeout) * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", fmt.Sprintf("%s:%d", config.Redis.Host, config.Redis.Port))
		},
	}
	return &Gores{pool: pool, prefix: PREFIX}
}

func (g *Gores) Close() error {
	return g.pool.Close()
}

func (g *Gores) Enqueue(jobData map[string]interface{}) error {
	job := GetJob()
	defer PutJob(job)

	job.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	job.Name = jobData["Name"].(string)
	job.Queue = jobData["Queue"].(string)
	for k, v := range jobData["Args"].(map[string]interface{}) {
		job.Args[k] = v
	}
	job.Retry = jobData["Retry"].(bool)
	job.EnqueueTime = float64(time.Now().Unix())

	if err := job.Validate(); err != nil {
		return err
	}
	data, err := job.ToBytes()
	if err != nil {
		return err
	}

	conn := g.pool.Get()
	defer conn.Close()

	queueKey := g.prefix + job.Queue + QUEUE_PENDING
	statKey := g.prefix + STAT_ENQUEUED
	script := redis.NewScript(2, luaEnqueue)
	_, err = script.Do(conn, queueKey, statKey, data)
	return err
}

func (g *Gores) EnqueueBatch(jobs []map[string]interface{}) error {
	if len(jobs) == 0 {
		return nil
	}
	conn := g.pool.Get()
	defer conn.Close()

	conn.Send("MULTI")
	for _, jobData := range jobs {
		job := GetJob()
		job.ID = fmt.Sprintf("%d", time.Now().UnixNano())
		job.Name = jobData["Name"].(string)
		job.Queue = jobData["Queue"].(string)
		for k, v := range jobData["Args"].(map[string]interface{}) {
			job.Args[k] = v
		}
		job.Retry = jobData["Retry"].(bool)
		job.EnqueueTime = float64(time.Now().Unix())

		if err := job.Validate(); err != nil {
			PutJob(job)
			return err
		}
		data, _ := job.ToBytes()
		conn.Send("LPUSH", g.prefix+job.Queue+QUEUE_PENDING, data)
		PutJob(job)
	}
	_, err := conn.Do("EXEC")
	return err
}

func (g *Gores) Info() (map[string]interface{}, error) {
	conn := g.pool.Get()
	defer conn.Close()

	conn.Send("MULTI")
	conn.Send("LLEN", g.prefix+"demo_queue"+QUEUE_PENDING)
	conn.Send("GET", g.prefix+STAT_ENQUEUED)
	conn.Send("GET", g.prefix+STAT_PROCESSED)
	results, err := redis.Values(conn.Do("EXEC"))
	if err != nil {
		return nil, err
	}

	pending, _ := redis.Int(results[0], nil)
	enqueued, _ := redis.Int(results[1], nil)
	processed, _ := redis.Int(results[2], nil)

	return map[string]interface{}{
		"pending":           pending,
		"enqueued":          enqueued,
		"processed":         processed,
		"Enqueue_timestamp": float64(time.Now().Unix()),
	}, nil
}
