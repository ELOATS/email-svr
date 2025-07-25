package queue

import (
	"context"
	"encoding/json"
	"time"

	"email-service/internal/mailer"

	"github.com/nats-io/nats.go"
)

// NATSQueue NATS队列实现
type NATSQueue struct {
	conn    *nats.Conn
	subject string
	sub     *nats.Subscription
	msgChan chan *nats.Msg
}

// NewNATSQueue 创建新的NATS队列
func NewNATSQueue(config *NATSConfig) (*NATSQueue, error) {
	conn, err := nats.Connect(config.URL)
	if err != nil {
		return nil, err
	}

	subject := config.Subject
	if subject == "" {
		subject = "email.jobs"
	}

	msgChan := make(chan *nats.Msg, 1000)

	// 创建订阅
	sub, err := conn.ChanSubscribe(subject, msgChan)
	if err != nil {
		conn.Close()
		return nil, err
	}

	return &NATSQueue{
		conn:    conn,
		subject: subject,
		sub:     sub,
		msgChan: msgChan,
	}, nil
}

// Push 将任务推入队列
func (n *NATSQueue) Push(ctx context.Context, job mailer.EmailJob) error {
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}

	return n.conn.Publish(n.subject, data)
}

// Pop 从队列中弹出任务
func (n *NATSQueue) Pop(ctx context.Context) (mailer.EmailJob, error) {
	var job mailer.EmailJob

	select {
	case msg := <-n.msgChan:
		err := json.Unmarshal(msg.Data, &job)
		if err != nil {
			return job, err
		}
		// 确认消息已处理
		err = msg.Ack()
		if err != nil {
			return mailer.EmailJob{}, err
		}
		return job, nil
	case <-ctx.Done():
		return job, ctx.Err()
	case <-time.After(1 * time.Second):
		// 超时返回，让调用者重试
		return job, ErrTimeout
	}
}

// Close 关闭连接
func (n *NATSQueue) Close() error {
	if n.sub != nil {
		err := n.sub.Unsubscribe()
		if err != nil {
			return err
		}
	}
	if n.conn != nil {
		n.conn.Close()
	}
	close(n.msgChan)
	return nil
}

// Size 返回队列中待处理任务数量（NATS不支持，返回-1）
func (n *NATSQueue) Size() (int, error) {
	return -1, ErrNotSupported
}
