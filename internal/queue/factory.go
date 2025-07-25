package queue

import (
	"fmt"

	"email-service/pkg/jobqueue"
)

// NewJobQueue 根据配置创建相应的队列实现
func NewJobQueue(config *TaskQueueConfig) (jobqueue.JobQueue, error) {
	switch config.Type {
	case TypeMemory:
		if config.Memory == nil {
			config.Memory = &MemoryConfig{BufferSize: 1000}
		}
		return NewMemoryQueue(config.Memory.BufferSize), nil
	case TypeRedis:
		if config.Redis == nil {
			return nil, ErrRedisConfigRequired
		}
		return NewRedisQueue(config.Redis)
	case TypeNATS:
		if config.NATS == nil {
			return nil, ErrNatsConfigRequired
		}
		return NewNATSQueue(config.NATS)
	case TypeKafka:
		if config.Kafka == nil {
			return nil, ErrKafkaConfigRequired
		}
		return NewKafkaQueue(config.Kafka)
	default:
		return nil, fmt.Errorf("unsupported queue type: %s (only memory supported currently)", config.Type)
	}
}
