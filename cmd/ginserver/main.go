package main

import (
	"log/slog"
	"os"

	"ollama_dev/internal/router"

	"github.com/gin-gonic/gin"
)

func main() {
	// 初始化日志工具
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// 初始化 Gin 引擎
	r := gin.Default()

	// 设置路由和中间件
	router.SetupRoutes(logger, r)

	// 启动 Gin 服务器
	logger.Info("Gin 服务器启动，监听端口 8080")
	if err := r.Run(":8080"); err != nil {
		logger.Error("服务器启动失败", "error", err)
	}
}
