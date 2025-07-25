package api

import (
	"email-service/internal/mailer"
)

// GlobalDispatcher 全局调度器实例
var GlobalDispatcher *mailer.Dispatcher

// SetDispatcher 设置全局调度器实例
func SetDispatcher(dispatcher *mailer.Dispatcher) {
	GlobalDispatcher = dispatcher
}
