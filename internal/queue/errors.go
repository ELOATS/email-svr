package queue

import "errors"

var (
	// ErrQueueFull 队列已满错误
	ErrQueueFull = errors.New("queue is full")
)

var (
	// ErrTimeout 超时错误
	ErrTimeout = errors.New("redis operation timeout")

	// ErrInvalidResult 无效结果错误
	ErrInvalidResult = errors.New("invalid redis result")

	// ErrRedisConfigRequired Redis配置是必须的
	ErrRedisConfigRequired = errors.New("redis config is required")
)

var (
	// ErrNatsTimeout 超时错误
	ErrNatsTimeout = errors.New("nats operation timeout")

	// ErrNotSupported 不支持的操作
	ErrNotSupported = errors.New("operation not supported by NATS queue")

	// ErrNatsConfigRequired NATS配置是必须的
	ErrNatsConfigRequired = errors.New("nats config is required")
)

var (
	ErrKafkaConfig         = errors.New("kafak config error")
	ErrKafkaConfigRequired = errors.New("kafka config is required")
)
