package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"

	"github.com/gorilla/websocket"
	"github.com/ollama/ollama/api"
	"github.com/patrickmn/go-cache"
	"github.com/tidwall/gjson"
)

// Logger 接口定义日志操作
type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}

// WSClient 接口定义 WebSocket 操作
type WSClient interface {
	Connect(url string) error
	ReadMessage() ([]byte, error)
	WriteMessage(message []byte) error
	Close() error
	Conn() *websocket.Conn // 新增接口方法
}

// OllamaClient 接口定义 Ollama 操作
type OllamaClient interface {
	Chat(modelName string, messages []api.Message) (string, error)
	ListModels() ([]map[string]string, error)
}

// Cache 接口定义缓存操作
type Cache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}, d time.Duration)
}

// MemoryCache 实现缓存
type MemoryCache struct {
	cache *cache.Cache
}

func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		cache: cache.New(120*time.Second, 10*time.Minute),
	}
}

func (m *MemoryCache) Get(key string) (interface{}, bool) {
	return m.cache.Get(key)
}

func (m *MemoryCache) Set(key string, value interface{}, d time.Duration) {
	m.cache.Set(key, value, d)
}

// WebSocketClient 实现 WSClient
type WebSocketClient struct {
	conn *websocket.Conn
}

func (w *WebSocketClient) Conn() *websocket.Conn {
	return w.conn
}

func NewWebSocketClient() *WebSocketClient {
	return &WebSocketClient{}
}

func (w *WebSocketClient) Connect(url string) error {
	header := make(http.Header)
	header.Add("Authorization", "Bearer valid-token")
	conn, _, err := websocket.DefaultDialer.Dial(url, header)
	if err != nil {
		return err
	}
	w.conn = conn
	return nil
}

func (w *WebSocketClient) ReadMessage() ([]byte, error) {
	_, message, err := w.conn.ReadMessage()
	return message, err
}

func (w *WebSocketClient) WriteMessage(message []byte) error {
	return w.conn.WriteMessage(websocket.TextMessage, message)
}

func (w *WebSocketClient) Close() error {
	return w.conn.Close()
}

// DefaultOllamaClient 实现 OllamaClient
type DefaultOllamaClient struct {
	client *api.Client
	cache  Cache
}

func NewOllamaClient(cache Cache) (*DefaultOllamaClient, error) {
	client, err := api.ClientFromEnvironment()
	if err != nil {
		return nil, err
	}
	return &DefaultOllamaClient{
		client: client,
		cache:  cache,
	}, nil
}

func (c *DefaultOllamaClient) Chat(modelName string, messages []api.Message) (string, error) {
	ctx := context.Background()
	req := &api.ChatRequest{
		Model:    modelName,
		Messages: messages,
		Stream:   new(bool),
	}

	var result string
	err := c.client.Chat(ctx, req, func(resp api.ChatResponse) error {
		result = resp.Message.Content
		return nil
	})

	return result, err
}

func (c *DefaultOllamaClient) ListModels() ([]map[string]string, error) {
	if cached, found := c.cache.Get("models"); found {
		return cached.([]map[string]string), nil
	}

	resp, err := c.client.List(context.Background())
	if err != nil {
		return nil, err
	}

	var data []map[string]string
	for _, model := range resp.Models {
		data = append(data, map[string]string{
			"model_name": model.Name,
			"status":     model.Digest,
		})
	}

	c.cache.Set("models", data, 120*time.Second)
	return data, nil
}

// HandlerFactory 请求处理器工厂
type HandlerFactory struct {
	ollamaClient OllamaClient
	logger       Logger
}

func NewHandlerFactory(ollamaClient OllamaClient, logger Logger) *HandlerFactory {
	return &HandlerFactory{
		ollamaClient: ollamaClient,
		logger:       logger,
	}
}

func (f *HandlerFactory) CreateHandler(action string) RequestHandler {
	switch action {
	case "list_model":
		return NewListModelHandler(f.ollamaClient, f.logger)
	case "chat":
		return NewChatHandler(f.ollamaClient, f.logger)
	default:
		return NewDefaultHandler(f.logger)
	}
}

// ChatHandler 实现
type ChatHandler struct {
	ollamaClient OllamaClient
	logger       Logger
}

func NewChatHandler(ollamaClient OllamaClient, logger Logger) *ChatHandler {
	return &ChatHandler{ollamaClient: ollamaClient, logger: logger}
}

func (h *ChatHandler) Handle(req *CloudRequest) (*CloudResponse, error) {
	var messages []api.Message
	for _, msg := range req.Params.Messages {
		messages = append(messages, api.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	response, err := h.ollamaClient.Chat(req.Params.ModelName, messages)
	if err != nil {
		return nil, err
	}

	return &CloudResponse{
		Type:      "client_to_server",
		Action:    req.Action,
		RequestID: req.RequestID,
		Data: map[string]interface{}{
			"message": map[string]string{
				"role":    "assistant",
				"content": response,
			},
		},
		Status: "done",
	}, nil
}

// RequestHandler 接口
type RequestHandler interface {
	Handle(req *CloudRequest) (*CloudResponse, error)
}

// DefaultHandler 实现
type DefaultHandler struct {
	logger Logger
}

func NewDefaultHandler(logger Logger) *DefaultHandler {
	return &DefaultHandler{logger: logger}
}

func (h *DefaultHandler) Handle(req *CloudRequest) (*CloudResponse, error) {
	return nil, fmt.Errorf("未知的动作: %s", req.Action)
}

// ListModelHandler 实现
type ListModelHandler struct {
	ollamaClient OllamaClient
	logger       Logger
}

func NewListModelHandler(ollamaClient OllamaClient, logger Logger) *ListModelHandler {
	return &ListModelHandler{ollamaClient: ollamaClient, logger: logger}
}

func (h *ListModelHandler) Handle(req *CloudRequest) (*CloudResponse, error) {
	models, err := h.ollamaClient.ListModels()
	if err != nil {
		return nil, err
	}

	return &CloudResponse{
		Type:      "client_to_server",
		Action:    req.Action,
		RequestID: req.RequestID,
		Data:      models,
		Status:    "done",
	}, nil
}

// Server 结构体
type Server struct {
	wsClient       WSClient
	handlerFactory *HandlerFactory
	logger         Logger
}

func NewServer(wsClient WSClient, handlerFactory *HandlerFactory, logger Logger) *Server {
	return &Server{
		wsClient:       wsClient,
		handlerFactory: handlerFactory,
		logger:         logger,
	}
}

const (
	heartbeatInterval = 30 * time.Second
	readTimeout       = 40 * time.Second
)

func (s *Server) Run() error {
	heartbeatTicker := time.NewTicker(heartbeatInterval)
	defer heartbeatTicker.Stop()

	for {
		// 设置读取超时
		if err := s.wsClient.Conn().SetReadDeadline(time.Now().Add(readTimeout)); err != nil {
			s.logger.Error("设置读取超时失败", "error", err)
			return err
		}

		select {
		case <-heartbeatTicker.C:
			if err := s.sendHeartbeat(); err != nil {
				s.logger.Error("发送心跳失败", "error", err)
				// 重连逻辑可以根据需要添加
			}

		default:
			msg, err := s.readAndParseMessage()
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					s.logger.Info("读取超时，等待下次心跳")
					continue
				}
				s.logger.Error("处理消息时发生错误", "error", err)
				continue // 不退出循环，继续处理后续消息
			}

			if msg.Response == nil {
				if err := s.handleServerRequest(msg); err != nil {
					s.logger.Error("处理服务端请求失败", "error", err)
				}
				continue
			}

			if err := s.processMessage(msg); err != nil {
				s.logger.Error("处理客户端响应失败", "error", err)
			}
		}
	}
}

func (s *Server) sendHeartbeat() error {
	requestID := uuid.New().String()
	heartbeatReq := &CloudRequest{
		Type:      "heartbeat",
		Action:    "ping",
		RequestID: requestID,
	}

	reqBytes, err := json.Marshal(heartbeatReq)
	if err != nil {
		return fmt.Errorf("心跳请求序列化失败: %w", err)
	}

	if err := s.wsClient.WriteMessage(reqBytes); err != nil {
		return fmt.Errorf("发送心跳消息失败: %w", err)
	}

	s.logger.Info("心跳已发送", "request_id", requestID)
	return nil
}

func (s *Server) handleServerRequest(msg *Message) error {
	if msg.Request == nil {
		return fmt.Errorf("处理消息时发生错误: 请求为空")
	}
	if msg.Request.Action == "" {
		return fmt.Errorf("处理消息时发生错误: 动作为空")
	}
	handler := s.handlerFactory.CreateHandler(msg.Request.Action)
	resp, err := handler.Handle(msg.Request)
	if err != nil {
		return err
	}
	msg.Response = resp
	return s.sendResponse(msg)
}

func (s *Server) readAndParseMessage() (*Message, error) {
	rawMsg, err := s.wsClient.ReadMessage()
	if err != nil {
		return nil, fmt.Errorf("WebSocket 读取消息错误: %w", err)
	}

	result := gjson.ParseBytes(rawMsg)

	if result.Get("request_id").Exists() {
		return &Message{
			Raw:      rawMsg,
			Response: &CloudResponse{},
		}, nil
	}

	req := &CloudRequest{
		Type:      result.Get("type").String(),
		Action:    result.Get("action").String(),
		RequestID: result.Get("request_id").String(),
	}

	return &Message{
		Raw:     rawMsg,
		Request: req,
	}, nil
}

func (s *Server) processMessage(msg *Message) error {
	handler := s.handlerFactory.CreateHandler(msg.Request.Action)
	resp, err := handler.Handle(msg.Request)
	if err != nil {
		return fmt.Errorf("处理请求失败: %w", err)
	}

	if resp == nil {
		return fmt.Errorf("处理请求失败: 响应为空")
	}

	msg.Response = resp
	return s.sendResponse(msg)
}

func (s *Server) sendResponse(msg *Message) error {
	respBytes, err := json.Marshal(msg.Response)
	if err != nil {
		return fmt.Errorf("JSON 序列化失败: %w", err)
	}

	if err := s.wsClient.WriteMessage(respBytes); err != nil {
		return fmt.Errorf("WebSocket 写入消息错误: %w", err)
	}

	return nil
}

// Message 结构体
type Message struct {
	Raw      []byte
	Request  *CloudRequest
	Response *CloudResponse
}

// CloudRequest 结构体
type CloudRequest struct {
	Type      string `json:"type"`
	Action    string `json:"action"`
	RequestID string `json:"request_id,omitempty"`
	Params    struct {
		ModelName string `json:"model_name,omitempty"`
		Messages  []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages,omitempty"`
	} `json:"params"`
}

// CloudResponse 结构体
type CloudResponse struct {
	Type      string `json:"type"`
	Action    string `json:"action"`
	RequestID string `json:"request_id,omitempty"`
	Data      any    `json:"data"`
	Status    string `json:"status,omitempty"`
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	logger.Info("请输入 WebSocket 地址 (例如 ws://localhost:8080/ws/ )")

	var serverAddr string
	_, _ = fmt.Scanln(&serverAddr)
	if serverAddr == "" {
		logger.Error("未提供有效的 WebSocket 地址")
		os.Exit(1)
	}

	var wsClient WSClient = NewWebSocketClient()

	// 连接重试逻辑
	connected := false
	for !connected {
		err := wsClient.Connect(serverAddr)
		if err == nil {
			connected = true
			continue
		}
		logger.Error("连接失败，正在重试...", "error", err)
		time.Sleep(5 * time.Second)
	}
	defer wsClient.Close()

	memoryCache := NewMemoryCache()
	ollamaClient, err := NewOllamaClient(memoryCache)
	if err != nil {
		logger.Error("创建Ollama客户端失败", "error", err)
		os.Exit(1)
	}

	handlerFactory := NewHandlerFactory(ollamaClient, logger)
	server := NewServer(wsClient, handlerFactory, logger)

	if err := server.Run(); err != nil {
		logger.Error("服务器运行错误", "error", err)
		os.Exit(1)
	}
}

func (s *Server) sendListModelRequest() error {
	requestID := uuid.New().String()
	request := &CloudRequest{
		Type:      "server_to_client",
		Action:    "list_model",
		RequestID: requestID,
	}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("JSON序列化失败: %w", err)
	}

	if err := s.wsClient.WriteMessage(requestBytes); err != nil {
		return fmt.Errorf("写入消息失败: %w", err)
	}

	s.logger.Info("已发送请求", "request_id", requestID)
	return nil
}
