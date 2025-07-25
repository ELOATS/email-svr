package config

import (
	"fmt"
	"os"
	"strconv"

	"email-service/internal/logger"
	"email-service/internal/queue"

	"github.com/spf13/viper"
)

// Config 存放所有应用配置
type Config struct {
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPass     string
	ServerPort   string
	MaxWorkers   int
	MaxQueueSize int
	Queue        *queue.TaskQueueConfig
	Logger       *logger.Config
}

// Load 从环境变量加载配置
func Load() (*Config, error) {
	smtpPort, err := strconv.Atoi(getEnv("SMTP_PORT", "587"))
	if err != nil {
		return nil, fmt.Errorf("invaild SMTP_PORT: %w", err)
	}
	maxWorkers, err := strconv.Atoi(getEnv("MAX_WORKERS", "10"))
	if err != nil {
		return nil, fmt.Errorf("invalid MAX_WORKERS: %w", err)
	}
	maxQueueSize, err := strconv.Atoi(getEnv("MAX_QUEUE_SIZE", "1000"))
	if err != nil {
		return nil, fmt.Errorf("invalid MAX_QUEUE_SIZE: %w", err)
	}
	// 默认内存队列配置
	queueConfig := &queue.TaskQueueConfig{
		Type: queue.TypeMemory,
		Memory: &queue.MemoryConfig{
			BufferSize: maxQueueSize,
		},
	}

	// 默认日志配置
	loggerConfig := &logger.Config{
		Level:     logger.LogLevel(getEnv("LOG_LEVEL", "info")),
		Format:    logger.LogFormat(getEnv("LOG_FORMAT", "text")),
		AddSource: getEnv("LOG_ADD_SOURCE", "false") == "true",
		Output:    getEnv("LOG_OUTPUT", "stdout"),
	}

	return &Config{
		SMTPHost:     getEnv("SMTP_HOST", "smtp.qq.com"),
		SMTPPort:     smtpPort,
		SMTPUser:     getEnv("SMTP_USER", "2514307815@qq.com"),
		SMTPPass:     getEnv("SMTP_PASS", ""),
		ServerPort:   getEnv("SERVER_PORT", "8080"),
		MaxWorkers:   maxWorkers,
		MaxQueueSize: maxQueueSize,
		Queue:        queueConfig,
		Logger:       loggerConfig,
	}, nil
}

// LoadWithViper 使用 viper 从配置文件或环境变量加载配置
func LoadWithViper(configPath string) (*Config, error) {
	v := viper.New()

	// 设置默认值
	v.SetDefault("smtp.host", "smtp.qq.com")
	v.SetDefault("smtp.port", 587)
	v.SetDefault("smtp.user", "2514307815@qq.com")
	v.SetDefault("smtp.pass", "")
	v.SetDefault("server.port", "8080")
	v.SetDefault("max_workers", 10)
	v.SetDefault("max_queue_size", 1000)
	v.SetDefault("logger.level", "info")
	v.SetDefault("logger.format", "text")
	v.SetDefault("logger.add_source", false)
	v.SetDefault("logger.output", "stdout")

	// 配置环境变量读取
	v.AutomaticEnv()
	v.SetEnvPrefix("EMAIL")
	_ = v.BindEnv("smtp.host", "SMTP_HOST")
	_ = v.BindEnv("smtp.port", "SMTP_PORT")
	_ = v.BindEnv("smtp.user", "SMTP_USER")
	_ = v.BindEnv("smtp.pass", "SMTP_PASS")
	_ = v.BindEnv("server.port", "SERVER_PORT")
	_ = v.BindEnv("max_workers", "MAX_WORKERS")
	_ = v.BindEnv("max_queue_size", "MAX_QUEUE_SIZE")
	_ = v.BindEnv("logger.level", "LOG_LEVEL")
	_ = v.BindEnv("logger.format", "LOG_FORMAT")
	_ = v.BindEnv("logger.add_source", "LOG_ADD_SOURCE")
	_ = v.BindEnv("logger.output", "LOG_OUTPUT")

	// 如果指定了配置文件路径，尝试读取配置文件
	if configPath != "" {
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// 解析队列配置
	var queueConfig queue.TaskQueueConfig
	if err := v.UnmarshalKey("queue", &queueConfig); err != nil {
		// 如果解析失败，使用默认内存队列配置
		queueConfig = queue.TaskQueueConfig{
			Type: queue.TypeMemory,
			Memory: &queue.MemoryConfig{
				BufferSize: v.GetInt("max_queue_size"),
			},
		}
	}

	// 解析日志配置
	var loggerConfig logger.Config
	if err := v.UnmarshalKey("logger", &loggerConfig); err != nil {
		// 如果解析失败，使用默认日志配置
		loggerConfig = logger.Config{
			Level:     logger.LogLevel(v.GetString("logger.level")),
			Format:    logger.LogFormat(v.GetString("logger.format")),
			AddSource: v.GetBool("logger.add_source"),
			Output:    v.GetString("logger.output"),
		}
	}

	return &Config{
		SMTPHost:     v.GetString("smtp.host"),
		SMTPPort:     v.GetInt("smtp.port"),
		SMTPUser:     v.GetString("smtp.user"),
		SMTPPass:     v.GetString("smtp.pass"),
		ServerPort:   v.GetString("server.port"),
		MaxWorkers:   v.GetInt("max_workers"),
		MaxQueueSize: v.GetInt("max_queue_size"),
		Queue:        &queueConfig,
		Logger:       &loggerConfig,
	}, nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
