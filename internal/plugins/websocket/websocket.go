package websocket

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request, logger *slog.Logger) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("WebSocket 升级失败", "error", err)
		return
	}
	client := &Client{Hub: hub, Conn: conn, Send: make(chan []byte, 256)}
	client.Hub.Register <- client
	go client.WritePump()
	go client.ReadPump()
}

func InitWebSocketPlugin(r *gin.RouterGroup, logger *slog.Logger) {
	h := NewHub()
	go h.Run()

	r.GET("/", func(c *gin.Context) {
		serveWs(h, c.Writer, c.Request, logger)
	})

	logger.Info("WebSocket 插件已加载，路径：/ws")
}
