package jobqueue

import (
	"context"
	"time"
)

// EmailJob 表示一个邮件发送任务
type EmailJob struct {
	To           string         `json:"to"`
	Subject      string         `json:"subject"`
	Body         string         `json:"body"`
	RetryCount   int            `json:"retry_count"`   // 当前重试次数
	MaxRetries   int            `json:"max_retries"`   // 最大重试次数
	NextRetryAt  time.Time      `json:"next_retry_at"` // 下次重试时间
	CreatedAt    time.Time      `json:"created_at"`    // 任务创建时间
	LastError    string         `json:"last_error"`    // 最后一次错误信息
	TemplateID   string         `json:"template_id"`   // 模板ID
	TemplateData map[string]any `json:"template_data"` // 模板数据
	Attachments  []Attachment   `json:"attachments"`   // 附件
}

// Attachment 附件
type Attachment struct {
	Filename string `json:"filename"` // 附件文件名
	Content  string `json:"content"`  // Base64编码的文件内容
	URL      string `json:"url"`      // 附件下载URL
}

// JobQueue 定义任务队列的接口
type JobQueue interface {
	// Push 将任务推入队列
	Push(ctx context.Context, job EmailJob) error

	// Pop 从队列中弹出任务（阻塞式）
	Pop(ctx context.Context) (EmailJob, error)

	// Close 关闭队列连接
	Close() error

	// Size 返回队列中待处理任务数量（可选实现，返回-1表示不支持）
	Size() (int, error)
}
