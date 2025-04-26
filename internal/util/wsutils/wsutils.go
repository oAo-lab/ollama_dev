package wsutils

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocket 消息类型
const (
	TextMessage   = websocket.TextMessage
	BinaryMessage = websocket.BinaryMessage
	CloseMessage  = websocket.CloseMessage
	PingMessage   = websocket.PingMessage
	PongMessage   = websocket.PongMessage
)

// Message 结构体，用于封装消息
type Message struct {
	Type int         `json:"type"` // 消息类型
	Data interface{} `json:"data"` // 消息数据
}

// Config 配置项
type Config struct {
	CheckOrigin func(r *http.Request) bool // 请求头校验函数
	Header      http.Header                // 自定义请求头
}

// WebSocketManager 管理 WebSocket 连接
type WebSocketManager struct {
	clients   map[*websocket.Conn]bool
	broadcast chan Message
	mu        sync.Mutex
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewWebSocketManager 创建一个新的 WebSocketManager
func NewWebSocketManager() *WebSocketManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &WebSocketManager{
		clients:   make(map[*websocket.Conn]bool),
		broadcast: make(chan Message),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Upgrade 升级 HTTP 连接为 WebSocket 连接
func (m *WebSocketManager) Upgrade(w http.ResponseWriter, r *http.Request, config Config) (*websocket.Conn, error) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			// 如果配置了自定义校验函数，则使用它
			if config.CheckOrigin != nil {
				return config.CheckOrigin(r)
			}
			// 默认允许所有来源
			return true
		},
	}

	// 添加自定义请求头
	if config.Header != nil {
		for key, values := range config.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, fmt.Errorf("升级 WebSocket 失败: %w", err)
	}

	// 注册客户端连接
	m.mu.Lock()
	m.clients[conn] = true
	m.mu.Unlock()

	// 启动心跳机制
	go m.startPingPong(conn)

	// 启动消息接收处理
	go m.receiveMessages(conn)

	return conn, nil
}

// SendMessage 发送消息到指定的 WebSocket 连接
func (m *WebSocketManager) SendMessage(conn *websocket.Conn, messageType int, data interface{}) error {
	payload, err := json.Marshal(Message{Type: messageType, Data: data})
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}
	return conn.WriteMessage(messageType, payload)
}

// Broadcast 广播消息到所有连接的客户端
func (m *WebSocketManager) Broadcast(messageType int, data interface{}) {
	msg := Message{Type: messageType, Data: data}
	m.broadcast <- msg
}

// Close 关闭 WebSocketManager
func (m *WebSocketManager) Close() {
	m.cancel()
	m.mu.Lock()
	defer m.mu.Unlock()

	for conn := range m.clients {
		conn.Close()
		delete(m.clients, conn)
	}
}

// startPingPong 启动心跳机制
func (m *WebSocketManager) startPingPong(conn *websocket.Conn) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	conn.SetPongHandler(func(string) error {
		log.Println("收到 Pong")
		return nil
	})

	for {
		select {
		case <-ticker.C:
			err := conn.WriteMessage(PingMessage, []byte{})
			if err != nil {
				log.Println("发送 Ping 失败:", err)
				return
			}
		case <-m.ctx.Done():
			return
		}
	}
}

func (m *WebSocketManager) StartPingPong(conn *websocket.Conn) {
	m.startPingPong(conn)
}

func (m *WebSocketManager) ReceiveMessages(conn *websocket.Conn) {
	m.receiveMessages(conn)
}

// receiveMessages 接收消息并处理
func (m *WebSocketManager) receiveMessages(conn *websocket.Conn) {
	defer func() {
		m.mu.Lock()
		delete(m.clients, conn)
		m.mu.Unlock()
		conn.Close()
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("读取消息失败:", err)
			return
		}

		// 处理消息
		var msg Message
		err = json.Unmarshal(message, &msg)
		if err != nil {
			log.Println("解析消息失败:", err)
			continue
		}

		// 根据消息类型进行处理
		switch msg.Type {
		case TextMessage:
			log.Printf("收到文本消息: %s", msg.Data)
		case BinaryMessage:
			log.Printf("收到二进制消息: %v", msg.Data)
		case CloseMessage:
			log.Println("收到关闭消息")
			return
		case PingMessage:
			log.Println("收到 Ping 消息")
			err := conn.WriteMessage(PongMessage, []byte{})
			if err != nil {
				log.Println("发送 Pong 失败:", err)
				return
			}
		case PongMessage:
			log.Println("收到 Pong 消息")
		default:
			log.Printf("未知消息类型: %d", msg.Type)
		}
	}
}

// ListenBroadcasts 监听广播消息并发送给所有客户端
func (m *WebSocketManager) ListenBroadcasts() {
	for {
		select {
		case msg := <-m.broadcast:
			m.mu.Lock()
			for client := range m.clients {
				err := client.WriteMessage(msg.Type, []byte(fmt.Sprintf("%v", msg.Data)))
				if err != nil {
					log.Println("广播消息失败:", err)
					client.Close()
					delete(m.clients, client)
				}
			}
			m.mu.Unlock()
		case <-m.ctx.Done():
			return
		}
	}
}
