package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// 定义 WebSocket 消息结构
type WebSocketMessage struct {
	Type             string `json:"type"`              // 消息类型，例如 "chat", "info" 等
	Content          string `json:"content"`           // 消息内容
	Group            string `json:"group"`             // 分组名称（如果是分组消息）
	Username         string `json:"username"`          // 当前用户名
	GroupSize        int    `json:"group_size"`        // 当前分组人数
	TotalConnections int    `json:"total_connections"` // 总连接数
}

// 定义 WebSocket 连接管理器
type ConnectionManager struct {
	connections map[*websocket.Conn]*ConnectionInfo
	groups      map[string]map[*websocket.Conn]bool // 分组管理：组名 -> 连接集合
	mu          sync.Mutex
}

// 连接信息结构
type ConnectionInfo struct {
	Username string // 用户名
	Group    string // 所属分组
}

// 初始化连接管理器
func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		connections: make(map[*websocket.Conn]*ConnectionInfo),
		groups:      make(map[string]map[*websocket.Conn]bool),
	}
}

// 添加连接
func (cm *ConnectionManager) AddConnection(conn *websocket.Conn, username string, group string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.connections[conn] = &ConnectionInfo{
		Username: username,
		Group:    group,
	}

	// 初始化分组（如果不存在）
	if _, exists := cm.groups[group]; !exists {
		cm.groups[group] = make(map[*websocket.Conn]bool)
	}
	cm.groups[group][conn] = true
}

// 移除连接
func (cm *ConnectionManager) RemoveConnection(conn *websocket.Conn) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if info, exists := cm.connections[conn]; exists {
		delete(cm.connections, conn)

		// 从分组中移除连接
		if conns, groupExists := cm.groups[info.Group]; groupExists {
			delete(conns, conn)
			if len(conns) == 0 {
				delete(cm.groups, info.Group) // 如果分组为空，则删除分组
			}
		}
	}
}

// 获取当前连接数
func (cm *ConnectionManager) GetTotalConnections() int {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	return len(cm.connections)
}

// 获取分组人数
func (cm *ConnectionManager) GetGroupSize(group string) int {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if conns, exists := cm.groups[group]; exists {
		return len(conns)
	}
	return 0
}

// 向指定分组广播消息
func (cm *ConnectionManager) BroadcastToGroup(group string, message *WebSocketMessage) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if conns, exists := cm.groups[group]; exists {
		for conn := range conns {
			cm.sendMessage(conn, message)
		}
	}
}

// 向单个连接发送消息
func (cm *ConnectionManager) sendMessage(conn *websocket.Conn, message *WebSocketMessage) {
	// 将消息序列化为 JSON
	msgBytes, err := json.Marshal(message)
	if err != nil {
		log.Println("json marshal error:", err)
		return
	}

	// 发送消息
	if err := conn.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
		log.Println("write error:", err)
		conn.Close()
		delete(cm.connections, conn)
	}
}

// 处理 WebSocket 连接
func handleWebSocketConnection(w http.ResponseWriter, r *http.Request, cm *ConnectionManager) {
	// 升级 HTTP 连接为 WebSocket 连接
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			// // 请求头校验：检查特定的请求头字段
			// if r.Header.Get("X-Custom-Header") != "expected-value" {
			// 	log.Println("Invalid custom header")
			// 	return false
			// }
			return true // 允许跨域请求
		},
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade error:", err)
		return
	}
	defer conn.Close()

	// 从请求中获取用户名和分组信息（假设通过查询参数传递）
	username := r.URL.Query().Get("username")
	group := r.URL.Query().Get("group")
	if username == "" || group == "" {
		log.Println("Username or group is missing")
		return
	}

	// 添加连接到管理器
	cm.AddConnection(conn, username, group)
	defer cm.RemoveConnection(conn)

	// 自动心跳维持
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				// 发送心跳消息
				cm.sendMessage(conn, &WebSocketMessage{
					Type:             "heartbeat",
					Content:          "ping",
					Username:         cm.connections[conn].Username,
					Group:            cm.connections[conn].Group,
					GroupSize:        cm.GetGroupSize(cm.connections[conn].Group),
					TotalConnections: cm.GetTotalConnections(),
				})
			}
		}
	}()

	// 循环读取客户端发送的消息
	for {
		_, messageBytes, err := conn.ReadMessage()
		if err != nil {
			log.Println("read error:", err)
			break
		}

		// 解析消息
		var receivedMessage WebSocketMessage
		if err := json.Unmarshal(messageBytes, &receivedMessage); err != nil {
			log.Println("json unmarshal error:", err)
			continue
		}

		// 触发消息处理
		switch receivedMessage.Type {
		case "join-group":
			// 更新分组信息
			cm.RemoveConnection(conn) // 先从旧分组移除
			cm.AddConnection(conn, receivedMessage.Username, receivedMessage.Group)
			cm.BroadcastToGroup(receivedMessage.Group, &WebSocketMessage{
				Type:             "group_update",
				Content:          receivedMessage.Username + " joined the group.",
				Username:         receivedMessage.Username,
				Group:            receivedMessage.Group,
				GroupSize:        cm.GetGroupSize(receivedMessage.Group),
				TotalConnections: cm.GetTotalConnections(),
			})
		case "leave-group":
			// 更新分组信息
			cm.BroadcastToGroup(receivedMessage.Group, &WebSocketMessage{
				Type:             "group_update",
				Content:          receivedMessage.Username + " left the group.",
				Username:         receivedMessage.Username,
				Group:            receivedMessage.Group,
				GroupSize:        cm.GetGroupSize(receivedMessage.Group),
				TotalConnections: cm.GetTotalConnections(),
			})
			cm.RemoveConnection(conn)
		default:
			// 默认回复消息，携带分组信息和连接信息
			cm.sendMessage(conn, &WebSocketMessage{
				Type:             "chat",
				Content:          receivedMessage.Content,
				Username:         cm.connections[conn].Username,
				Group:            cm.connections[conn].Group,
				GroupSize:        cm.GetGroupSize(cm.connections[conn].Group),
				TotalConnections: cm.GetTotalConnections(),
			})
		}
	}
}

func main() {
	// 初始化连接管理器
	cm := NewConnectionManager()

	// 注册 WebSocket 处理函数
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handleWebSocketConnection(w, r, cm)
	})

	// 启动 HTTP 服务器，监听 8080 端口
	log.Println("Server started at :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("ListenAndServe error:", err)
	}
}
