package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"email-service/pkg/jobqueue"

	"github.com/segmentio/kafka-go"
)

// KafkaQueue 定义了Kafka队列
type KafkaQueue struct {
	writer *kafka.Writer // 写入器
	reader *kafka.Reader // 读取器
	topic  string        // 主题
}

// NewKafkaQueue 创建Kafka队列
func NewKafkaQueue(cfg *KafkaConfig) (*KafkaQueue, error) {
	if len(cfg.Brokers) == 0 || cfg.Topic == "" || cfg.GroupID == "" {
		return nil, ErrKafkaConfigRequired
	}

	writer := &kafka.Writer{
		Addr:     kafka.TCP(cfg.Brokers...),
		Topic:    cfg.Topic,
		Balancer: &kafka.LeastBytes{},
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  cfg.Brokers,
		Topic:    cfg.Topic,
		GroupID:  cfg.GroupID,
		MinBytes: 10e3, // 10KB，最小消息
		MaxBytes: 10e6, // 10MB，最大消息
	})

	return &KafkaQueue{
		writer: writer,
		reader: reader,
		topic:  cfg.Topic,
	}, nil
}

// Push 将任务推送到Kafka队列
func (k *KafkaQueue) Push(ctx context.Context, job jobqueue.EmailJob) error {
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}
	msg := kafka.Message{
		Value: data,
		Time:  time.Now(),
	}
	return k.writer.WriteMessages(ctx, msg)
}

// Pop 从Kafka队列中获取任务
func (k *KafkaQueue) Pop(ctx context.Context) (jobqueue.EmailJob, error) {
	m, err := k.reader.ReadMessage(ctx)
	if err != nil {
		return jobqueue.EmailJob{}, err
	}
	var job jobqueue.EmailJob
	if err := json.Unmarshal(m.Value, &job); err != nil {
		return jobqueue.EmailJob{}, err
	}
	return job, nil
}

// Close 关闭Kafka队列
func (k *KafkaQueue) Close() error {
	err1 := k.writer.Close()
	err2 := k.reader.Close()
	if err1 != nil {
		return err1
	}
	return err2
}

func (k *KafkaQueue) Size() (int, error) {
	// Kafka 不支持直接获取队列长度，返回-1
	return -1, nil
}
