package mailer

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"time"

	"email-service/internal/logger"
	"email-service/pkg/jobqueue"

	"gopkg.in/gomail.v2"
)

// RetryScheduler 定义重试调度接口
type RetryScheduler interface {
	ScheduleRetry(job *jobqueue.EmailJob, err error)
}

// Worker 负责从任务队列中取出并处理邮件任务
type Worker struct {
	ID             int
	dialer         *gomail.Dialer
	jobQueue       jobqueue.JobQueue      // 任务队列
	retryManager   *jobqueue.RetryManager // 重试管理器
	retryScheduler RetryScheduler         // 重试调度器
	ctx            context.Context
	logger         *logger.Logger
}

// NewWorker 创建一个新的工人实例
func NewWorker(id int, dialer *gomail.Dialer, jobQueue jobqueue.JobQueue, retryManager *jobqueue.RetryManager, ctx context.Context) *Worker {
	return &Worker{
		ID:           id,
		dialer:       dialer,
		jobQueue:     jobQueue,
		retryManager: retryManager,
		ctx:          ctx,
		logger:       logger.GetDefault().WithWorker(id),
	}
}

// SetRetryScheduler 设置重试调度器
func (w *Worker) SetRetryScheduler(scheduler RetryScheduler) {
	w.retryScheduler = scheduler
}

// Start 启动工人，使其开始监听任务
func (w *Worker) Start() {
	go w.processJobs()
}

// processJobs 处理任务队列中的任务
func (w *Worker) processJobs() {
	for {
		select {
		case <-w.ctx.Done():
			w.logger.Info("Worker shutting down")
			return
		default:
			job, err := w.jobQueue.Pop(w.ctx)
			if err != nil {
				// 如果是超时或上下文取消，继续循环
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					continue
				}
				// 其他错误，等待一段时间后重试
				w.logger.Debug("Failed to pop job from queue", "error", err)
				time.Sleep(time.Second)
				continue
			}

			// 检查是否到了重试时间
			if !w.retryManager.IsReadyForRetry(&job) {
				// 还没到重试时间，重新放回队列
				delay := time.Until(job.NextRetryAt)
				w.logger.Debug("Job not ready for retry, delaying",
					"recipient", job.To,
					"delay", delay.Round(time.Second))

				// 延迟后重新入队
				time.AfterFunc(delay, func() {
					if err = w.jobQueue.Push(w.ctx, job); err != nil {
						w.logger.Error("Failed to requeue delayed job",
							"recipient", job.To,
							"error", err)
					}
				})
				continue
			}

			// 处理任务
			w.processJob(job)
		}
	}
}

// processJob 处理单个邮件发送任务
func (w *Worker) processJob(job jobqueue.EmailJob) {
	startTime := time.Now()

	// ====== 增加日志记录，追踪任务处理开始 ======
	w.logger.Info("Starting to process job", "to", job.To)
	// ====== end ======

	retryInfo := w.retryManager.GetRetryInfo(&job)

	w.logger.Info("Processing email",
		"recipient", job.To,
		"subject", job.Subject,
		"retry_info", retryInfo,
		"created_at", job.CreatedAt)

	// ====== 新增模板渲染逻辑 ======
	if job.TemplateID != "" {
		tmplPath := fmt.Sprintf("templates/%s", job.TemplateID)
		tmpl, err := template.ParseFiles(tmplPath)
		if err != nil {
			w.logger.Error("Failed to parse template", "template", tmplPath, "error", err)
			job.Body = "<h1>Failed to parse template</h1>"
		}
		var buf bytes.Buffer
		err = tmpl.Execute(&buf, job.TemplateData)
		if err != nil {
			w.logger.Error("Failed to execute template", "template", tmplPath, "error", err)
			job.Body = "<h1>Failed to execute template</h1>"
		}
		job.Body = buf.String()
	}
	// ====== end ======

	// 创建邮件
	m := gomail.NewMessage()
	m.SetHeader("From", w.dialer.Username)
	m.SetHeader("To", job.To)
	m.SetHeader("Subject", job.Subject)
	m.SetBody("text/html", job.Body)

	// 处理附件
	w.processAttachments(&job, m)

	// 发送邮件
	err := w.sendEmail(m)
	duration := time.Since(startTime)

	if err != nil {
		w.logger.Error("Failed to send email", "to", job.To, "duration", duration, "error", err)
		w.retryScheduler.ScheduleRetry(&job, err)
		return
	}

	w.logger.Info("Successfully sent email", "to", job.To, "duration", duration)
}

// sendEmail 封装了实际的邮件发送逻辑
func (w *Worker) sendEmail(m *gomail.Message) error {
	// DialAndSend 会处理连接、认证和发送的整个过程
	return w.dialer.DialAndSend(m)
}

// processAttachments 处理附件
func (w *Worker) processAttachments(job *jobqueue.EmailJob, m *gomail.Message) {
	for _, att := range job.Attachments {
		data, err := w.getAttachmentData(&att)
		if err != nil {
			w.logger.Error("Failed to get attachment data", "filename", att.Filename, "url", att.URL, "error", err)
			continue
		}

		if len(data) > 0 {
			m.Attach(att.Filename, gomail.SetCopyFunc(func(writer io.Writer) error {
				_, err = writer.Write(data)
				return err
			}))
		}
	}
}

// getAttachmentData 根据附件定义获取附件数据
func (w *Worker) getAttachmentData(att *jobqueue.Attachment) ([]byte, error) {
	if att.URL != "" {
		return w.downloadAttachment(att.URL)
	}
	if att.Content != "" {
		return base64.StdEncoding.DecodeString(att.Content)
	}
	return nil, nil // 没有附件内容
}

// downloadAttachment 从 URL 下载附件
func (w *Worker) downloadAttachment(url string) ([]byte, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get(url)
	if err != nil {
		w.logger.Error("Failed to download attachment", "url", url, "error", err)
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		if err = Body.Close(); err != nil {
			w.logger.Error("Failed to close attachment body", "url", url, "error", err)
			return
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		w.logger.Error("Failed to download attachment", "url", url, "status_code", resp.StatusCode)
		return nil, errors.New("failed to download attachment")
	}

	return io.ReadAll(resp.Body)
}
