package queue

import (
	"context"

	"email-service/pkg/jobqueue"
)

// MemoryQueue 内存队列实现
type MemoryQueue struct {
	jobChan chan jobqueue.EmailJob
}

// NewMemoryQueue 创建新的内存队列
func NewMemoryQueue(bufferSize int) *MemoryQueue {
	return &MemoryQueue{
		jobChan: make(chan jobqueue.EmailJob, bufferSize),
	}
}

// Push 将任务推入队列
func (m *MemoryQueue) Push(ctx context.Context, job jobqueue.EmailJob) error {
	select {
	case m.jobChan <- job:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		// 非阻塞模式，如果队列满了就返回错误
		return ErrQueueFull
	}
}

// Pop 从队列中弹出任务
func (m *MemoryQueue) Pop(ctx context.Context) (jobqueue.EmailJob, error) {
	select {
	case job := <-m.jobChan:
		return job, nil
	case <-ctx.Done():
		return jobqueue.EmailJob{}, ctx.Err()
	}
}

// Close 关闭队列
func (m *MemoryQueue) Close() error {
	close(m.jobChan)
	return nil
}

// Size 返回队列中待处理任务数量
func (m *MemoryQueue) Size() (int, error) {
	return len(m.jobChan), nil
}
