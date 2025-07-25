// Package queue 队列配置定义
package queue

// TaskQueueType 队列类型
type TaskQueueType string

const (
	TypeMemory TaskQueueType = "memory"
	TypeRedis  TaskQueueType = "redis"
	TypeNATS   TaskQueueType = "nats"
	TypeKafka  TaskQueueType = "kafka"
)

// TaskQueueConfig 队列配置
type TaskQueueConfig struct {
	Type   TaskQueueType `mapstructure:"type"`
	Redis  *RedisConfig  `mapstructure:"redis,omitempty"`
	NATS   *NATSConfig   `mapstructure:"nats,omitempty"`
	Memory *MemoryConfig `mapstructure:"memory,omitempty"`
	Kafka  *KafkaConfig  `mapstructure:"kafka,omitempty"`
}

// RedisConfig Redis队列配置
type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	QueueKey string `mapstructure:"queue_key"`
}

// NATSConfig NATS队列配置
type NATSConfig struct {
	URL     string `mapstructure:"url"`
	Subject string `mapstructure:"subject"`
}

// MemoryConfig 内存队列配置
type MemoryConfig struct {
	BufferSize int `mapstructure:"buffer_size"`
}

// KafkaConfig Kafka队列配置
type KafkaConfig struct {
	Brokers []string `mapstructure:"brokers"`  // Kafka集群地址
	Topic   string   `mapstructure:"topic"`    // Kafka主题
	GroupID string   `mapstructure:"group_id"` // Kafka消费者组ID
}
