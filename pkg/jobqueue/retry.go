package jobqueue

import (
	"fmt"
	"math"
	"time"
)

// RetryConfig 重试配置
type RetryConfig struct {
	MaxRetries      int           `json:"max_retries"`      // 最大重试次数
	BaseDelay       time.Duration `json:"base_delay"`       // 基础延迟时间
	MaxDelay        time.Duration `json:"max_delay"`        // 最大延迟时间
	BackoffMultiple float64       `json:"backoff_multiple"` // 退避倍数
	JitterEnabled   bool          `json:"jitter_enabled"`   // 是否启用抖动
}

// DefaultRetryConfig 默认重试配置
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:      3,
		BaseDelay:       1 * time.Minute,
		MaxDelay:        30 * time.Minute,
		BackoffMultiple: 2.0,
		JitterEnabled:   true,
	}
}

// RetryManager 重试管理器
type RetryManager struct {
	config *RetryConfig
}

// NewRetryManager 创建新的重试管理器
func NewRetryManager(config *RetryConfig) *RetryManager {
	if config == nil {
		config = DefaultRetryConfig()
	}
	return &RetryManager{
		config: config,
	}
}

// ShouldRetry 判断是否应该重试
func (rm *RetryManager) ShouldRetry(job *EmailJob) bool {
	return job.RetryCount < job.MaxRetries
}

// CalculateNextRetryDelay 计算下次重试的延迟时间（指数退避算法）
func (rm *RetryManager) CalculateNextRetryDelay(retryCount int) time.Duration {
	// 指数退避: baseDelay * (backoffMultiple ^ retryCount)
	delay := float64(rm.config.BaseDelay) * math.Pow(rm.config.BackoffMultiple, float64(retryCount))

	// 限制最大延迟时间
	if delay > float64(rm.config.MaxDelay) {
		delay = float64(rm.config.MaxDelay)
	}

	duration := time.Duration(delay)

	// 添加抖动，避免雷群效应
	if rm.config.JitterEnabled {
		jitter := time.Duration(float64(duration) * 0.1) // 10%的抖动
		duration += time.Duration((time.Now().UnixNano() % int64(jitter*2)) - int64(jitter))
	}

	return duration
}

// PrepareRetry 准备重试，更新任务的重试信息
func (rm *RetryManager) PrepareRetry(job *EmailJob, err error) *EmailJob {
	job.RetryCount++
	job.LastError = err.Error()

	// 计算下次重试时间
	delay := rm.CalculateNextRetryDelay(job.RetryCount - 1)
	job.NextRetryAt = time.Now().Add(delay)

	return job
}

// CreateJobWithRetry 创建带重试配置的任务
func (rm *RetryManager) CreateJobWithRetry(to, subject, body string) *EmailJob {
	return &EmailJob{
		To:          to,
		Subject:     subject,
		Body:        body,
		RetryCount:  0,
		MaxRetries:  rm.config.MaxRetries,
		NextRetryAt: time.Now(),
		CreatedAt:   time.Now(),
		LastError:   "",
	}
}

// IsReadyForRetry 检查任务是否已到重试时间
func (rm *RetryManager) IsReadyForRetry(job *EmailJob) bool {
	return time.Now().After(job.NextRetryAt) || time.Now().Equal(job.NextRetryAt)
}

// GetRetryInfo 获取重试信息字符串
func (rm *RetryManager) GetRetryInfo(job *EmailJob) string {
	if job.RetryCount == 0 {
		return "First attempt"
	}

	timeUntilRetry := time.Until(job.NextRetryAt)
	if timeUntilRetry <= 0 {
		return fmt.Sprintf("Retry %d/%d (Ready)", job.RetryCount, job.MaxRetries)
	}

	return fmt.Sprintf("Retry %d/%d (Next retry in %v)", job.RetryCount, job.MaxRetries, timeUntilRetry.Round(time.Second))
}
