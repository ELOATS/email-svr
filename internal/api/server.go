package api

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RunServer(port string) {
	http.HandleFunc("/v1/send-event-email", HandleSendNotification)

	log.Printf("HTTP server starting on port %s...", port)

	addr := fmt.Sprintf(":%s", port)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("FATAL: HTTP server failed to start: %v", err)
	}
}

func RunGinServer(port string) {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	r.POST("/v1/send-event-email", SendEmailHandler)
	r.POST("/v1/preview-template", PreviewTemplateHandler)

	addr := fmt.Sprintf(":%s", port)
	if err := r.Run(addr); err != nil {
		return
	}
}
