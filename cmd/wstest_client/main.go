package main

import (
	"bufio"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/websocket"

	"ollama_dev/internal/util/wsutils"
)

func main() {
	// 自定义 Dialer，设置 Origin 请求头
	dialer := websocket.Dialer{}

	// 设置自定义请求头
	header := http.Header{}
	header.Add("Origin", "http://allowed-origin.com") // 设置为服务端允许的 Origin
	header.Add("X-Custom-Header", "ClientValue")      // 自定义请求头

	// 连接到 WebSocket 服务器
	conn, resp, err := dialer.Dial("ws://localhost:8080/ws", header)
	if err != nil {
		log.Fatalf("连接失败: %v, 响应: %v", err, resp)
	}
	defer conn.Close()

	log.Println("已连接到服务器")

	m := wsutils.NewWebSocketManager()

	m.ReceiveMessages(conn)

	go m.StartPingPong(conn)

	// 发送消息
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		text := scanner.Text()
		err := conn.WriteMessage(websocket.TextMessage, []byte(text))
		if err != nil {
			log.Println("发送消息失败:", err)
			return
		}
		log.Printf("已发送消息: %s", text)
	}
}
