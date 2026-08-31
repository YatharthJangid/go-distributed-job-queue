package gores

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/garyburd/redigo/redis"
)

type Gores struct {
	pool   *redis.Pool
	prefix string
}

const luaEnqueue = `
	local queue = KEYS[1]
	local statKey = KEYS[2]
	local data = ARGV[1]
	redis.call('LPUSH', queue, data)
	redis.call('INCR', statKey)
	return 1
`

const luaEnqueueIdempotent = `
	local queue = KEYS[1]
	local statKey = KEYS[2]
	local idempKey = KEYS[3]
	local dupStatKey = KEYS[4]
	local data = ARGV[1]
	local ttl = tonumber(ARGV[2])

	local setRes = redis.call('SET', idempKey, 'enqueued', 'EX', ttl, 'NX')
	if not setRes then
		redis.call('INCR', dupStatKey)
		return 0
	end

	redis.call('LPUSH', queue, data)
	redis.call('INCR', statKey)
	return 1
`

const luaEnqueueBatch = `
	local statKey = KEYS[1]
	local dupStatKey = KEYS[2]
	local prefix = ARGV[1]
	local numJobs = tonumber(ARGV[2])
	local enqueuedCount = 0
	local dupCount = 0

	for i = 1, numJobs do
		local queueKey = ARGV[2 + (i-1)*4 + 1]
		local data = ARGV[2 + (i-1)*4 + 2]
		local idempKey = ARGV[2 + (i-1)*4 + 3]
		local ttl = tonumber(ARGV[2 + (i-1)*4 + 4])

		local shouldEnqueue = true
		if idempKey ~= "" then
			local fullKey = prefix .. idempKey
			local setRes = redis.call('SET', fullKey, 'enqueued', 'EX', ttl, 'NX')
			if not setRes then
				shouldEnqueue = false
				dupCount = dupCount + 1
			end
		end

		if shouldEnqueue then
			redis.call('LPUSH', queueKey, data)
			enqueuedCount = enqueuedCount + 1
		end
	end

	if enqueuedCount > 0 then
		redis.call('INCRBY', statKey, enqueuedCount)
	end
	if dupCount > 0 then
		redis.call('INCRBY', dupStatKey, dupCount)
	end
	return enqueuedCount
`

func dialRedisURL(rawurl string) (redis.Conn, error) {
	u, err := url.Parse(rawurl)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "redis" && u.Scheme != "rediss" {
		return nil, fmt.Errorf("invalid redis URL scheme: %s", u.Scheme)
	}

	host, port, err := net.SplitHostPort(u.Host)
	if err != nil {
		host = u.Host
		port = "6379"
	}
	if host == "" {
		host = "localhost"
	}
	address := net.JoinHostPort(host, port)

	var options []redis.DialOption
	if u.Scheme == "rediss" {
		options = append(options, redis.DialUseTLS(true))
	}

	conn, err := redis.Dial("tcp", address, options...)
	if err != nil {
		return nil, err
	}

	if u.User != nil {
		username := u.User.Username()
		password, hasPassword := u.User.Password()
		if username != "" && hasPassword {
			if _, err := conn.Do("AUTH", username, password); err != nil {
				conn.Close()
				return nil, err
			}
		} else if hasPassword {
			if _, err := conn.Do("AUTH", password); err != nil {
				conn.Close()
				return nil, err
			}
		}
	}

	if u.Path != "" && u.Path != "/" {
		dbStr := strings.TrimPrefix(u.Path, "/")
		if db, err := strconv.Atoi(dbStr); err == nil && db != 0 {
			if _, err := conn.Do("SELECT", db); err != nil {
				conn.Close()
				return nil, err
			}
		}
	}

	return conn, nil
}

func NewGores(config *Config) *Gores {
	pool := &redis.Pool{
		MaxIdle:     config.Redis.MaxIdle,
		MaxActive:   config.Redis.MaxActive,
		IdleTimeout: time.Duration(config.Redis.IdleTimeout) * time.Second,
		Dial: func() (redis.Conn, error) {
			if config.Redis.URL != "" {
				return dialRedisURL(config.Redis.URL)
			}
			return redis.Dial("tcp", fmt.Sprintf("%s:%d", config.Redis.Host, config.Redis.Port))
		},
	}
	return &Gores{pool: pool, prefix: PREFIX}
}

func (g *Gores) Close() error {
	return g.pool.Close()
}

func jobFromMap(jobData map[string]interface{}) *Job {
	job := GetJob()
	if id, ok := jobData["ID"].(string); ok && id != "" {
		job.ID = id
	} else {
		job.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	}

	if key, ok := jobData["IdempotencyKey"].(string); ok {
		job.IdempotencyKey = key
	}

	if ttl, ok := jobData["IdempotencyTTL"].(int); ok && ttl > 0 {
		job.IdempotencyTTL = ttl
	} else if ttlF, ok := jobData["IdempotencyTTL"].(float64); ok && ttlF > 0 {
		job.IdempotencyTTL = int(ttlF)
	} else if job.IdempotencyKey != "" {
		job.IdempotencyTTL = DEFAULT_IDEMPOTENCY_TTL
	}

	job.Name = jobData["Name"].(string)
	job.Queue = jobData["Queue"].(string)
	if args, ok := jobData["Args"].(map[string]interface{}); ok {
		for k, v := range args {
			job.Args[k] = v
		}
	}
	if retry, ok := jobData["Retry"].(bool); ok {
		job.Retry = retry
	}
	job.EnqueueTime = float64(time.Now().Unix())
	return job
}

func (g *Gores) Enqueue(jobData map[string]interface{}) error {
	job := jobFromMap(jobData)
	defer PutJob(job)

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

	if job.IdempotencyKey != "" {
		idempKey := g.prefix + IDEMPOTENCY_PREFIX + job.IdempotencyKey
		dupStatKey := g.prefix + STAT_DUPLICATES
		script := redis.NewScript(4, luaEnqueueIdempotent)
		_, err = script.Do(conn, queueKey, statKey, idempKey, dupStatKey, data, job.IdempotencyTTL)
		return err
	}

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

	args := make([]interface{}, 0, 4+len(jobs)*4)
	statKey := g.prefix + STAT_ENQUEUED
	dupStatKey := g.prefix + STAT_DUPLICATES

	// Script keys: statKey, dupStatKey (2 keys)
	// Script args: prefix, numJobs, followed by [queueKey, data, idempKey, ttl] per job
	args = append(args, statKey, dupStatKey, g.prefix+IDEMPOTENCY_PREFIX, len(jobs))

	for _, jobData := range jobs {
		job := jobFromMap(jobData)

		if err := job.Validate(); err != nil {
			PutJob(job)
			return err
		}
		data, _ := job.ToBytes()
		queueKey := g.prefix + job.Queue + QUEUE_PENDING

		args = append(args, queueKey, data, job.IdempotencyKey, job.IdempotencyTTL)
		PutJob(job)
	}

	script := redis.NewScript(2, luaEnqueueBatch)
	_, err := script.Do(conn, args...)
	return err
}

func (g *Gores) Info() (map[string]interface{}, error) {
	conn := g.pool.Get()
	defer conn.Close()

	conn.Send("MULTI")
	conn.Send("LLEN", g.prefix+"demo_queue"+QUEUE_PENDING)
	conn.Send("GET", g.prefix+STAT_ENQUEUED)
	conn.Send("GET", g.prefix+STAT_PROCESSED)
	conn.Send("GET", g.prefix+STAT_DUPLICATES)
	results, err := redis.Values(conn.Do("EXEC"))
	if err != nil {
		return nil, err
	}

	pending, _ := redis.Int(results[0], nil)
	enqueued, _ := redis.Int(results[1], nil)
	processed, _ := redis.Int(results[2], nil)
	duplicates, _ := redis.Int(results[3], nil)

	return map[string]interface{}{
		"pending":           pending,
		"enqueued":          enqueued,
		"processed":         processed,
		"duplicates":        duplicates,
		"Enqueue_timestamp": float64(time.Now().Unix()),
	}, nil
}

func (g *Gores) GetJobStatus(idempotencyKey string) (string, error) {
	if idempotencyKey == "" {
		return "", fmt.Errorf("idempotencyKey empty")
	}
	conn := g.pool.Get()
	defer conn.Close()

	// Check execution key first
	execKey := g.prefix + EXEC_PREFIX + idempotencyKey
	execStatus, err := redis.String(conn.Do("GET", execKey))
	if err == nil && execStatus != "" {
		return execStatus, nil
	}

	// Check enqueue idempotency key
	idempKey := g.prefix + IDEMPOTENCY_PREFIX + idempotencyKey
	idempStatus, err := redis.String(conn.Do("GET", idempKey))
	if err == nil && idempStatus != "" {
		return idempStatus, nil
	}

	return "unknown", nil
}
