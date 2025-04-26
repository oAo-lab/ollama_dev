package router

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"ollama_dev/internal/middleware"
	"ollama_dev/internal/plugins/websocket"
)

// SetupRoutes 注册路由
func SetupRoutes(logger *slog.Logger, r *gin.Engine) {
	// 全局中间件
	r.Use(middleware.CorsMiddleware())
	r.Use(middleware.TrafficLoggingMiddleware(logger))
	// r.Use(middleware.AuthMiddleware())

	logger.Info("中间件已加载")

	// WebSocket 插件路由组
	wsGroup := r.Group("/ws")
	{
		websocket.InitWebSocketPlugin(wsGroup, logger)
	}
}
