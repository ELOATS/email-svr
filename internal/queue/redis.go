package queue

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"email-service/internal/mailer"

	"github.com/redis/go-redis/v9"
)

// RedisQueue Redis队列实现
type RedisQueue struct {
	client   *redis.Client
	queueKey string
}

// NewRedisQueue 创建新的Redis队列
func NewRedisQueue(config *RedisConfig) (*RedisQueue, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     config.Addr,
		Password: config.Password,
		DB:       config.DB,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	queueKey := config.QueueKey
	if queueKey == "" {
		queueKey = "email:jobs"
	}

	return &RedisQueue{
		client:   client,
		queueKey: queueKey,
	}, nil
}

// Push 将任务推入队列
func (r *RedisQueue) Push(ctx context.Context, job mailer.EmailJob) error {
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}

	return r.client.LPush(ctx, r.queueKey, data).Err()
}

// Pop 从队列中弹出任务（阻塞式）
func (r *RedisQueue) Pop(ctx context.Context) (mailer.EmailJob, error) {
	var job mailer.EmailJob

	// 使用BRPOP进行阻塞式弹出，超时时间设为1秒
	result, err := r.client.BRPop(ctx, 1*time.Second, r.queueKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			// 超时，返回空任务和nil错误，让调用者重试
			return job, ErrTimeout
		}
		return job, err
	}

	// result[0]是key, result[1]是value
	if len(result) < 2 {
		return job, ErrInvalidResult
	}

	err = json.Unmarshal([]byte(result[1]), &job)
	return job, err
}

// PopNonBlocking 从队列中弹出任务（非阻塞式）
func (r *RedisQueue) PopNonBlocking(ctx context.Context) (mailer.EmailJob, error) {
	var job mailer.EmailJob

	result, err := r.client.RPop(ctx, r.queueKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			// 队列为空，返回空任务和nil错误，让调用者重试
			return job, ErrTimeout
		}
		return job, err
	}

	err = json.Unmarshal([]byte(result), &job)
	return job, err
}

// Close 关闭连接
func (r *RedisQueue) Close() error {
	return r.client.Close()
}

// Size 返回队列中待处理任务数量
func (r *RedisQueue) Size() (int, error) {
	ctx := context.Background()
	length, err := r.client.LLen(ctx, r.queueKey).Result()
	return int(length), err
}
