package lib

import (
	"fmt"
	"sync"

	"github.com/vmihailenco/msgpack/v5"
)

type Job struct {
	ID          string                 `msgpack:"id"`
	Name        string                 `msgpack:"name"`
	Queue       string                 `msgpack:"queue"`
	Args        map[string]interface{} `msgpack:"args"`
	Retry       bool                   `msgpack:"retry"`
	RetryCount  int                    `msgpack:"retry_count"`
	EnqueueTime float64                `msgpack:"enqueue_time"`
}

var jobPool = sync.Pool{
	New: func() interface{} {
		return &Job{Args: make(map[string]interface{}, 8)}
	},
}

func GetJob() *Job {
	return jobPool.Get().(*Job)
}

func PutJob(j *Job) {
	j.ID, j.Name, j.Queue = "", "", ""
	for k := range j.Args {
		delete(j.Args, k)
	}
	j.Retry, j.RetryCount = false, 0
	jobPool.Put(j)
}

func (j *Job) ToBytes() ([]byte, error) {
	return msgpack.Marshal(j)
}

func FromBytes(data []byte) (*Job, error) {
	var j Job
	if err := msgpack.Unmarshal(data, &j); err != nil {
		return nil, fmt.Errorf("msgpack: %w", err)
	}
	return &j, nil
}

func (j *Job) Validate() error {
	if j.Name == "" || j.Queue == "" {
		return fmt.Errorf("name/queue empty")
	}
	return nil
}
