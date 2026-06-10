package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"go-database/internal/connection"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// wsHub manages active WebSocket connections
type wsHub struct {
	mu      sync.RWMutex
	clients map[string]map[*websocket.Conn]bool
}

var hub = &wsHub{clients: make(map[string]map[*websocket.Conn]bool)}

func (h *wsHub) join(connID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[connID] == nil {
		h.clients[connID] = make(map[*websocket.Conn]bool)
	}
	h.clients[connID][conn] = true
}

func (h *wsHub) leave(connID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[connID] != nil {
		delete(h.clients[connID], conn)
		if len(h.clients[connID]) == 0 {
			delete(h.clients, connID)
		}
	}
}

func (h *wsHub) broadcast(connID string, msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for conn := range h.clients[connID] {
		conn.WriteMessage(websocket.TextMessage, msg)
	}
}

// wsQueryMsg is the JSON format for WebSocket query messages
type wsQueryMsg struct {
	Type    string `json:"type"`    // "query" | "execute" | "ping"
	Query   string `json:"query,omitempty"`
	ReqID   string `json:"req_id,omitempty"`
}

// wsRespMsg is sent back to the client
type wsRespMsg struct {
	Type    string          `json:"type"`
	ReqID   string          `json:"req_id,omitempty"`
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// WSQueryHandler creates a Gin handler for streaming query WebSocket
func WSQueryHandler(connMgr *connection.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		connID := c.Param("id")
		username, _ := c.Get("username")

		ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			slog.Error("websocket upgrade failed", "err", err)
			return
		}
		defer ws.Close()

		hub.join(connID, ws)
		defer hub.leave(connID, ws)

		slog.Info("ws connected", "conn_id", connID, "user", username)

		// send welcome
		hub.broadcast(connID, mustJSON(wsRespMsg{Type: "connected", Success: true}))

		for {
			_, msg, err := ws.ReadMessage()
			if err != nil {
				slog.Info("ws disconnected", "conn_id", connID, "user", username)
				break
			}

			var req wsQueryMsg
			if err := json.Unmarshal(msg, &req); err != nil {
				hub.broadcast(connID, mustJSON(wsRespMsg{Type: "error", Error: "invalid JSON"}))
				continue
			}

			switch req.Type {
			case "ping":
				hub.broadcast(connID, mustJSON(wsRespMsg{Type: "pong", ReqID: req.ReqID, Success: true}))

			case "query":
				go func(r wsQueryMsg) {
					ctx := c.Request.Context()
					start := time.Now()
					result, err := connMgr.Query(ctx, connID, r.Query)
					if err != nil {
						hub.broadcast(connID, mustJSON(wsRespMsg{
							Type: "result", ReqID: r.ReqID, Success: false,
							Error: fmt.Sprintf("query failed: %s", err),
						}))
						return
					}
					data, _ := json.Marshal(result)
					hub.broadcast(connID, mustJSON(wsRespMsg{
						Type: "result", ReqID: r.ReqID, Success: true, Data: data,
					}))
					slog.Debug("ws query", "conn_id", connID, "user", username,
						"duration", time.Since(start), "req_id", r.ReqID)
				}(req)

			case "execute":
				go func(r wsQueryMsg) {
					ctx := c.Request.Context()
					start := time.Now()
					result, err := connMgr.Execute(ctx, connID, r.Query)
					if err != nil {
						hub.broadcast(connID, mustJSON(wsRespMsg{
							Type: "result", ReqID: r.ReqID, Success: false,
							Error: fmt.Sprintf("execute failed: %s", err),
						}))
						return
					}
					data, _ := json.Marshal(result)
					hub.broadcast(connID, mustJSON(wsRespMsg{
						Type: "result", ReqID: r.ReqID, Success: true, Data: data,
					}))
					slog.Debug("ws execute", "conn_id", connID, "user", username,
						"duration", time.Since(start), "req_id", r.ReqID)
				}(req)

			default:
				hub.broadcast(connID, mustJSON(wsRespMsg{
					Type: "error", Error: fmt.Sprintf("unknown type: %s", req.Type),
				}))
			}
		}
	}
}

// NotifyWebSocket sends a notification to all clients watching a connection
func NotifyWebSocket(connID string, event string, data any) {
	msg, _ := json.Marshal(map[string]any{
		"type": "notification",
		"event": event,
		"data":  data,
		"time":  time.Now().UTC(),
	})
	hub.broadcast(connID, msg)
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
