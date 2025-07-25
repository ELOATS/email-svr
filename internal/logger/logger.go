package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"time"
)

// LogLevel 日志级别类型
type LogLevel string

const (
	LevelDebug LogLevel = "debug"
	LevelInfo  LogLevel = "info"
	LevelWarn  LogLevel = "warn"
	LevelError LogLevel = "error"
)

// LogFormat 日志格式类型
type LogFormat string

const (
	FormatJSON LogFormat = "json"
	FormatText LogFormat = "text"
)

// Config 日志配置
type Config struct {
	Level     LogLevel  `json:"level" yaml:"level" mapstructure:"level"`
	Format    LogFormat `json:"format" yaml:"format" mapstructure:"format"`
	AddSource bool      `json:"add_source" yaml:"add_source" mapstructure:"add_source"`
	Output    string    `json:"output" yaml:"output" mapstructure:"output"` // "stdout", "stderr", 或文件路径
}

// DefaultConfig 返回默认日志配置
func DefaultConfig() *Config {
	return &Config{
		Level:     LevelInfo,
		Format:    FormatText,
		AddSource: false,
		Output:    "stdout",
	}
}

// Logger 包装slog.Logger，提供额外功能
type Logger struct {
	*slog.Logger
	config *Config
}

var defaultLogger *Logger

// Init 初始化全局日志器
func Init(config *Config) error {
	if config == nil {
		config = DefaultConfig()
	}

	var writer io.Writer
	switch config.Output {
	case "stdout", "":
		writer = os.Stdout
	case "stderr":
		writer = os.Stderr
	default:
		// 文件输出
		file, err := os.OpenFile(config.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return err
		}
		writer = file
	}

	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level:     parseLogLevel(config.Level),
		AddSource: config.AddSource,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// 自定义时间格式
			if a.Key == slog.TimeKey {
				return slog.String("timestamp", a.Value.Time().Format(time.RFC3339))
			}
			return a
		},
	}

	switch config.Format {
	case FormatJSON:
		handler = slog.NewJSONHandler(writer, opts)
	default:
		handler = slog.NewTextHandler(writer, opts)
	}

	logger := slog.New(handler)
	defaultLogger = &Logger{
		Logger: logger,
		config: config,
	}

	// 设置为全局默认日志器
	slog.SetDefault(logger)

	return nil
}

// parseLogLevel 解析日志级别
func parseLogLevel(level LogLevel) slog.Level {
	switch level {
	case LevelDebug:
		return slog.LevelDebug
	case LevelInfo:
		return slog.LevelInfo
	case LevelWarn:
		return slog.LevelWarn
	case LevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// GetDefault 获取默认日志器
func GetDefault() *Logger {
	if defaultLogger == nil {
		// 如果没有初始化，使用默认配置
		err := Init(nil)
		if err != nil {
			return nil
		}
	}
	return defaultLogger
}

// WithContext 创建带上下文的日志器
func (l *Logger) WithContext(ctx context.Context) *Logger {
	return &Logger{
		Logger: l.Logger,
		config: l.config,
	}
}

// WithWorker 创建带Worker信息的日志器
func (l *Logger) WithWorker(workerID int) *Logger {
	return &Logger{
		Logger: l.Logger.With("worker_id", workerID),
		config: l.config,
	}
}

// WithJob 创建带任务信息的日志器
func (l *Logger) WithJob(jobID, recipient string) *Logger {
	return &Logger{
		Logger: l.Logger.With("job_id", jobID, "recipient", recipient),
		config: l.config,
	}
}

// WithRetry 创建带重试信息的日志器
func (l *Logger) WithRetry(retryCount, maxRetries int) *Logger {
	return &Logger{
		Logger: l.Logger.With("retry_count", retryCount, "max_retries", maxRetries),
		config: l.config,
	}
}

// WithComponent 创建带组件信息的日志器
func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{
		Logger: l.Logger.With("component", component),
		config: l.config,
	}
}

// WithError 创建带错误信息的日志器
func (l *Logger) WithError(err error) *Logger {
	return &Logger{
		Logger: l.Logger.With("error", err),
		config: l.config,
	}
}

// LogEmailSent 记录邮件发送成功日志
func (l *Logger) LogEmailSent(workerID int, recipient string, duration time.Duration, retryCount int) {
	logger := l.WithWorker(workerID)
	if retryCount > 0 {
		logger.Info("Email sent successfully after retries",
			"recipient", recipient,
			"duration", duration,
			"retry_count", retryCount,
		)
	} else {
		logger.Info("Email sent successfully",
			"recipient", recipient,
			"duration", duration,
		)
	}
}

// LogEmailFailed 记录邮件发送失败日志
func (l *Logger) LogEmailFailed(workerID int, recipient string, err error, retryCount int, willRetry bool) {
	logger := l.WithWorker(workerID).WithError(err)
	if willRetry {
		logger.Warn("Email sending failed, will retry",
			"recipient", recipient,
			"retry_count", retryCount,
		)
	} else {
		logger.Error("Email sending failed permanently",
			"recipient", recipient,
			"retry_count", retryCount,
		)
	}
}

// LogRetryScheduled 记录重试调度日志
func (l *Logger) LogRetryScheduled(recipient string, retryCount, maxRetries int, delay time.Duration) {
	l.Info("Retry scheduled",
		"recipient", recipient,
		"retry_count", retryCount,
		"max_retries", maxRetries,
		"delay", delay,
	)
}

// LogQueueOperation 记录队列操作日志
func (l *Logger) LogQueueOperation(operation string, queueType string, size int) {
	l.Debug("Queue operation",
		"operation", operation,
		"queue_type", queueType,
		"queue_size", size,
	)
}

// Info Convenience functions for global logger
func Info(msg string, args ...any) {
	GetDefault().Info(msg, args...)
}

func Debug(msg string, args ...any) {
	GetDefault().Debug(msg, args...)
}

func Warn(msg string, args ...any) {
	GetDefault().Warn(msg, args...)
}

func Error(msg string, args ...any) {
	GetDefault().Error(msg, args...)
}

func Fatal(msg string, args ...any) {
	GetDefault().Error(msg, args...)
	os.Exit(1)
}

func With(args ...any) *Logger {
	return &Logger{
		Logger: GetDefault().With(args...),
		config: GetDefault().config,
	}
}
