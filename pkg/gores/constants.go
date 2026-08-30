package gores

const (
	PREFIX                 = "gores:"
	QUEUE_PENDING          = ":pending"
	QUEUE_PROCESS          = ":processing"
	QUEUE_DELAYED          = ":delayed"
	STAT_ENQUEUED          = "stat:enqueued"
	STAT_PROCESSED         = "stat:processed"
	STAT_DUPLICATES        = "stat:duplicates"
	IDEMPOTENCY_PREFIX     = "idempotency:"
	EXEC_PREFIX            = "exec:"
	DEFAULT_IDEMPOTENCY_TTL = 3600
)
