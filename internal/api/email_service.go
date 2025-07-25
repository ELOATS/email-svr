package api

import (
	"email-service/internal/logger"
	"email-service/internal/mailer"
	"time"
)

// EmailService 邮件服务
type EmailService struct {
	dispatcher MailDispatcher
	logger     *logger.Logger
}

// MailDispatcher 定义了邮件作业分发器的接口
type MailDispatcher interface {
	PushJob(job mailer.EmailJob) error
}

// NewEmailService 创建一个新的邮件服务
func NewEmailService(dispatcher MailDispatcher) *EmailService {
	return &EmailService{
		dispatcher: dispatcher,
		logger:     logger.GetDefault().WithComponent("email-service"),
	}
}

func (s *EmailService) QueueEmailJobs(req SendEmailRequest) (int, []error) {
	var errs []error
	var successfulCount int

	for _, email := range req.Recipients {
		job := mailer.EmailJob{
			To:           email,
			Subject:      req.Subject,
			MaxRetries:   3,
			NextRetryAt:  time.Now(),
			CreatedAt:    time.Now(),
			Attachments:  req.Attachments,
			TemplateID:   req.TemplateID,
			TemplateData: req.TemplateData,
		}

		if err := s.dispatcher.PushJob(job); err != nil {
			s.logger.Error("Failed to push job to queue", "recipient", email, "error", err)
			errs = append(errs, err)
		} else {
			successfulCount++
		}
	}
	return successfulCount, errs
}
