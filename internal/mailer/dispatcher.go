package mailer

import (
	"context"
	"time"

	"email-service/internal/logger"
	"email-service/pkg/jobqueue"

	"gopkg.in/gomail.v2"
)

// Dispatcher 负责管理工人和任务分发
type Dispatcher struct {
	dialer       *gomail.Dialer
	maxWorkers   int                    // 最大工人数量
	jobQueue     jobqueue.JobQueue      // 任务队列
	retryManager *jobqueue.RetryManager // 重试管理器
	ctx          context.Context
	cancel       context.CancelFunc
	logger       *logger.Logger
}

// NewDispatcher 创建一个新的调度器
func NewDispatcher(dialer *gomail.Dialer, maxWorkers int, jobQueue jobqueue.JobQueue) *Dispatcher {
	ctx, cancel := context.WithCancel(context.Background())
	return &Dispatcher{
		dialer:       dialer,
		maxWorkers:   maxWorkers,
		jobQueue:     jobQueue,
		retryManager: jobqueue.NewRetryManager(nil), // 使用默认重试配置
		ctx:          ctx,
		cancel:       cancel,
		logger:       logger.GetDefault().WithComponent("dispatcher"),
	}
}

// Run 启动调度器，创建并运行所有工人
func (d *Dispatcher) Run() {
	for i := 1; i <= d.maxWorkers; i++ {
		worker := NewWorker(i, d.dialer, d.jobQueue, d.retryManager, d.ctx)
		worker.SetRetryScheduler(d) // 设置调度器作为重试调度器
		worker.Start()
	}
	d.logger.Info("Workers started and ready to process jobs", "worker_count", d.maxWorkers)
}

// Stop 停止调度器和所有工人
func (d *Dispatcher) Stop() {
	d.logger.Info("Stopping dispatcher and all workers")
	d.cancel()
	if err := d.jobQueue.Close(); err != nil {
		d.logger.Error("Error closing job queue", "error", err)
	}
}

// PushJob 将任务推入队列
func (d *Dispatcher) PushJob(job EmailJob) error {
	return d.jobQueue.Push(d.ctx, job)
}

// ScheduleRetry 安排任务重试
func (d *Dispatcher) ScheduleRetry(job *jobqueue.EmailJob, err error) {
	if !d.retryManager.ShouldRetry(job) {
		d.logger.Error("Task failed permanently",
			"recipient", job.To,
			"retry_count", job.RetryCount,
			"error", err)
		return
	}

	// 准备重试
	retryJob := d.retryManager.PrepareRetry(job, err)

	// 计算延迟时间
	delay := time.Until(retryJob.NextRetryAt)

	d.logger.LogRetryScheduled(retryJob.To, retryJob.RetryCount, retryJob.MaxRetries, delay)

	// 使用定时器安排重试
	time.AfterFunc(delay, func() {
		if err := d.jobQueue.Push(d.ctx, *retryJob); err != nil {
			d.logger.Error("Failed to reschedule retry",
				"recipient", retryJob.To,
				"retry_count", retryJob.RetryCount,
				"error", err)
		} else {
			d.logger.Debug("Retry rescheduled successfully",
				"recipient", retryJob.To,
				"retry_count", retryJob.RetryCount,
				"max_retries", retryJob.MaxRetries)
		}
	})
}
