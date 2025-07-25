package main

import (
	"crypto/tls"
	"log"

	"email-service/internal/api"
	"email-service/internal/config"
	"email-service/internal/mailer"
	"email-service/internal/queue"

	"gopkg.in/gomail.v2"
)

func main() {
	var cfg *config.Config
	var err error

	// 检查是否指定了配置文件
	configFile := "local.yaml"
	// 使用Viper从配置文件加载
	cfg, err = config.LoadWithViper(configFile)
	if err != nil {
		log.Fatalf("FATAL: Could not load config from file %s: %v", configFile, err)
	}
	log.Printf("Configuration loaded from file: %s", configFile)

	// 连接SMTP服务器
	dialer := gomail.NewDialer(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass)
	dialer.TLSConfig = &tls.Config{
		ServerName:         cfg.SMTPHost,
		InsecureSkipVerify: false,
	}
	if d, err2 := dialer.Dial(); err2 != nil {
		log.Fatalf("FATAL: Failed to connect to SMTP server: %v", err2)
	} else {
		if err = d.Close(); err != nil {
			log.Fatalf("FATAL: Failed to close SMTP server connection: %v", err)
			return
		}
	}
	log.Println("SMTP server connection verified.")

	// 创建队列实例
	jobQueue, err := queue.NewJobQueue(cfg.Queue)
	if err != nil {
		log.Fatalf("FATAL: Failed to create job queue: %v", err)
	}
	log.Printf("Job queue created: type=%s", cfg.Queue.Type)

	// 创建调度器
	dispatcher := mailer.NewDispatcher(dialer, cfg.MaxWorkers, jobQueue)
	// 启动调度器
	dispatcher.Run()

	// 设置全局调度器
	api.SetDispatcher(dispatcher)

	// 启动 API 服务
	api.RunGinServer(cfg.ServerPort)
}
