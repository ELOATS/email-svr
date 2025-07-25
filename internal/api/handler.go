package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"email-service/internal/logger"
	"email-service/internal/mailer"
	"email-service/pkg/jobqueue"

	"github.com/gin-gonic/gin"
)

// NotificationPayload 是 API 接收的请求体
type NotificationPayload struct {
	Subject    string   `json:"subject"`
	Body       string   `json:"body"`
	Recipients []string `json:"recipients"` // 接收者邮箱列表
}

// HandleSendNotification 处理邮件发送请求
func HandleSendNotification(w http.ResponseWriter, r *http.Request) {
	apiLogger := logger.GetDefault().WithComponent("api")

	if r.Method != http.MethodPost {
		apiLogger.Warn("Invalid HTTP method",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr)
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload NotificationPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		apiLogger.Error("Invalid request body",
			"error", err,
			"remote_addr", r.RemoteAddr)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 通过全局调度器推送任务到队列
	for _, email := range payload.Recipients {
		job := mailer.EmailJob{
			To:          email,
			Subject:     payload.Subject,
			Body:        payload.Body,
			RetryCount:  0,
			MaxRetries:  3, // 默认最多重试3次
			NextRetryAt: time.Now(),
			CreatedAt:   time.Now(),
			LastError:   "",
		}

		if err := GlobalDispatcher.PushJob(job); err != nil {
			apiLogger.Error("Failed to push job to queue",
				"recipient", email,
				"error", err)
		}
	}

	apiLogger.Info("Email jobs queued successfully",
		"job_count", len(payload.Recipients),
		"subject", payload.Subject,
		"remote_addr", r.RemoteAddr)

	// 返回 202 Accepted，表示请求已接收，正在异步处理
	w.WriteHeader(http.StatusAccepted)
	err := json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Jobs accepted for processing.",
		"count":   len(payload.Recipients),
	})
	if err != nil {
		apiLogger.Error("Failed to encode response",
			"error", err,
			"remote_addr", r.RemoteAddr)
		return
	}

}

// SendEmailRequest 用于 Gin 版本的邮件发送接口
// 支持模板和附件
type SendEmailRequest struct {
	Subject      string                `json:"subject" binding:"required"`
	Recipients   []string              `json:"recipients" binding:"required"`
	TemplateID   string                `json:"template_id"`
	TemplateData map[string]any        `json:"template_data"`
	Attachments  []jobqueue.Attachment `json:"attachments"`
}

// SendEmailHandler 基于 Gin 的邮件发送接口
func SendEmailHandler(c *gin.Context) {
	apiLogger := logger.GetDefault().WithComponent("api")

	var req SendEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apiLogger.Error("Invalid request body", "error", err, "remote_addr", c.ClientIP())
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	emailService := NewEmailService(GlobalDispatcher)
	successfulCount, errs := emailService.QueueEmailJobs(req)

	apiLogger.Info("Email jobs queued successfully (gin)",
		"job_count", len(req.Recipients),
		"successful_count", successfulCount,
		"failed_count", len(errs),
		"subject", req.Subject,
		"remote_addr", c.ClientIP())

	if len(errs) > 0 {
		apiLogger.Error("Failed to queue email jobs", "errors", errs, "remote_addr", c.ClientIP())
		c.JSON(http.StatusInternalServerError, gin.H{
			"message":          "Some jobs were accepted, but failures occurred.",
			"successful_count": successfulCount,
			"failed_count":     len(errs),
		})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Jobs accepted for processing.",
		"count":   len(req.Recipients),
	})
}

type PreviewTemplateRequest struct {
	TemplateID   string         `json:"template_id" binding:"required"`
	TemplateData map[string]any `json:"template_data"`
}

// PreviewTemplateHandler 预览模板
func PreviewTemplateHandler(c *gin.Context) {
	var req PreviewTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	tmplPath := fmt.Sprintf("templates/%s", req.TemplateID)
	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse template"})
		return
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, req.TemplateData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to execute template"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"html": buf.String()})
}
