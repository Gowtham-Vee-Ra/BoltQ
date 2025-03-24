// internal/api/websocket.go
package api

import (
	"BoltQ/internal/job"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"BoltQ/pkg/logger"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Allow all origins for development; in production, restrict this
	CheckOrigin: func(r *http.Request) bool { return true },
}

// WebSocketManager handles WebSocket connections and real-time updates
type WebSocketManager struct {
	redisClient     *redis.Client
	logger          *logger.Logger
	clients         map[*websocket.Conn]bool
	broadcast       chan []byte
	register        chan *websocket.Conn
	unregister      chan *websocket.Conn
	ctx             context.Context
	cancel          context.CancelFunc
	jobChannel      string
	workflowChannel string
	mu              sync.Mutex
}

// NewWebSocketManager creates a new WebSocket manager
func NewWebSocketManager(client *redis.Client, logger *logger.Logger) *WebSocketManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &WebSocketManager{
		redisClient:     client,
		logger:          logger,
		clients:         make(map[*websocket.Conn]bool),
		broadcast:       make(chan []byte),
		register:        make(chan *websocket.Conn),
		unregister:      make(chan *websocket.Conn),
		ctx:             ctx,
		cancel:          cancel,
		jobChannel:      "job_updates",
		workflowChannel: "workflow_updates",
	}
}

// Start begins the WebSocket manager
func (wm *WebSocketManager) Start() {
	go wm.run()
	go wm.subscribeToRedis()
}

// Stop gracefully shuts down the WebSocket manager
func (wm *WebSocketManager) Stop() {
	wm.cancel()

	// Close all client connections
	wm.mu.Lock()
	for client := range wm.clients {
		client.Close()
	}
	wm.mu.Unlock()
}

// run handles WebSocket events
func (wm *WebSocketManager) run() {
	for {
		select {
		case client := <-wm.register:
			wm.mu.Lock()
			wm.clients[client] = true
			wm.mu.Unlock()
			wm.logger.Info("New WebSocket client connected")

		case client := <-wm.unregister:
			wm.mu.Lock()
			if _, ok := wm.clients[client]; ok {
				delete(wm.clients, client)
				client.Close()
			}
			wm.mu.Unlock()
			wm.logger.Info("WebSocket client disconnected")

		case message := <-wm.broadcast:
			wm.mu.Lock()
			for client := range wm.clients {
				wm.sendMessage(client, message)
			}
			wm.mu.Unlock()

		case <-wm.ctx.Done():
			return
		}
	}
}

// sendMessage sends a message to a specific client
func (wm *WebSocketManager) sendMessage(client *websocket.Conn, message []byte) {
	client.SetWriteDeadline(time.Now().Add(writeWait))

	w, err := client.NextWriter(websocket.TextMessage)
	if err != nil {
		client.Close()
		return
	}

	w.Write(message)

	if err := w.Close(); err != nil {
		client.Close()
		return
	}
}

// subscribeToRedis subscribes to Redis PubSub channels for updates
func (wm *WebSocketManager) subscribeToRedis() {
	pubsub := wm.redisClient.Subscribe(wm.ctx, wm.jobChannel, wm.workflowChannel)
	defer pubsub.Close()

	ch := pubsub.Channel()

	for {
		select {
		case msg := <-ch:
			wm.broadcast <- []byte(msg.Payload)
		case <-wm.ctx.Done():
			return
		}
	}
}

// HandleJobUpdatesWebSocket handles WebSocket connections for job updates
func (wm *WebSocketManager) HandleJobUpdatesWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		wm.logger.Error(fmt.Sprintf("Error upgrading connection to WebSocket: %v", err))
		return
	}

	// Register the client
	wm.register <- conn

	// Unregister client when the function returns
	defer func() {
		wm.unregister <- conn
	}()

	// Set up connection parameters
	conn.SetReadLimit(maxMessageSize)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	// Listen for messages from the client (not used but needed to keep connection alive)
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				wm.logger.Error(fmt.Sprintf("WebSocket error: %v", err))
			}
			break
		}
	}
}

// PublishJobUpdate publishes a job update to all connected clients
func (wm *WebSocketManager) PublishJobUpdate(jobID, status string, data map[string]interface{}) error {
	message := map[string]interface{}{
		"type":      "job_update",
		"job_id":    jobID,
		"status":    status,
		"data":      data,
		"timestamp": time.Now(),
	}

	jsonMessage, err := json.Marshal(message)
	if err != nil {
		return err
	}

	return wm.redisClient.Publish(wm.ctx, wm.jobChannel, string(jsonMessage)).Err()
}

// PublishWorkflowUpdate publishes a workflow update to all connected clients
func (wm *WebSocketManager) PublishWorkflowUpdate(workflowID string, status job.WorkflowStatus, data map[string]interface{}) error {
	message := map[string]interface{}{
		"type":        "workflow_update",
		"workflow_id": workflowID,
		"status":      status,
		"data":        data,
		"timestamp":   time.Now(),
	}

	jsonMessage, err := json.Marshal(message)
	if err != nil {
		return err
	}

	return wm.redisClient.Publish(wm.ctx, wm.workflowChannel, string(jsonMessage)).Err()
}
