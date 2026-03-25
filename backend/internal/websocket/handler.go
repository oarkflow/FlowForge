package websocket

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"time"

	fws "github.com/fasthttp/websocket"
	"github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp"

	"github.com/oarkflow/deploy/backend/internal/db/queries"
)

// Handler manages WebSocket connections for real-time log streaming.
type Handler struct {
	hub      *Hub
	repos    *queries.Repositories
	upgrader fws.FastHTTPUpgrader
}

// NewHandler creates a new WebSocket handler.
func NewHandler(hub *Hub, repos *queries.Repositories) *Handler {
	return &Handler{
		hub:   hub,
		repos: repos,
		upgrader: fws.FastHTTPUpgrader{
			CheckOrigin: func(ctx *fasthttp.RequestCtx) bool {
				return true // CORS is handled at middleware level
			},
		},
	}
}

// HandleRunLogs handles WebSocket connections for streaming pipeline run logs.
// URL: /ws/runs/:runId/logs
func (h *Handler) HandleRunLogs(c fiber.Ctx) error {
	runID := c.Params("runId")
	if runID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "missing run_id")
	}

	return h.upgrader.Upgrade(c.RequestCtx(), func(conn *fws.Conn) {
		defer conn.Close()

		client := &Client{
			RunID: runID,
			Send:  make(chan []byte, 256),
		}

		h.hub.Register <- client
		defer func() {
			h.hub.Unregister <- client
		}()

		// Replay existing logs on connect
		h.replayLogs(conn, runID)

		// Write pump: send messages from hub to WebSocket
		done := make(chan struct{})
		go func() {
			defer close(done)
			for msg := range client.Send {
				if err := conn.WriteMessage(fws.TextMessage, msg); err != nil {
					return
				}
			}
		}()

		// Read pump: keep connection alive until client disconnects
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}

		<-done
	})
}

// HandleEvents handles WebSocket connections for general pipeline events.
// URL: /ws/events
func (h *Handler) HandleEvents(c fiber.Ctx) error {
	return h.upgrader.Upgrade(c.RequestCtx(), func(conn *fws.Conn) {
		defer conn.Close()

		client := &Client{
			RunID: "__events__",
			Send:  make(chan []byte, 256),
		}

		h.hub.Register <- client
		defer func() {
			h.hub.Unregister <- client
		}()

		done := make(chan struct{})
		go func() {
			defer close(done)
			for msg := range client.Send {
				if err := conn.WriteMessage(fws.TextMessage, msg); err != nil {
					return
				}
			}
		}()

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}

		<-done
	})
}

// replayLogs sends all existing logs for a run to the WebSocket client.
func (h *Handler) replayLogs(conn *fws.Conn, runID string) {
	if h.repos == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	logs, err := h.repos.Logs.GetByRunID(ctx, runID, 10000, 0)
	if err != nil {
		return
	}

	for _, logEntry := range logs {
		stepRunID := ""
		if logEntry.StepRunID != nil {
			stepRunID = *logEntry.StepRunID
		}

		msg, _ := json.Marshal(map[string]string{
			"type":        "log",
			"run_id":      logEntry.RunID,
			"step_run_id": stepRunID,
			"stream":      logEntry.Stream,
			"content":     logEntry.Content,
		})

		if err := conn.WriteMessage(fws.TextMessage, msg); err != nil {
			return
		}
	}

	marker, _ := json.Marshal(map[string]string{
		"type":   "replay_complete",
		"run_id": runID,
	})
	_ = conn.WriteMessage(fws.TextMessage, marker)
}

// SSEHandler provides a Server-Sent Events fallback for clients that don't
// support WebSocket connections.
// URL: /sse/runs/:runId/logs
func (h *Handler) SSEHandler(c fiber.Ctx) error {
	runID := c.Params("runId")
	if runID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "missing run_id")
	}

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")

	client := &Client{
		RunID: runID,
		Send:  make(chan []byte, 256),
	}

	h.hub.Register <- client
	defer func() {
		h.hub.Unregister <- client
	}()

	c.RequestCtx().SetBodyStreamWriter(func(w *bufio.Writer) {
		for {
			select {
			case msg, ok := <-client.Send:
				if !ok {
					return
				}
				_, _ = fmt.Fprintf(w, "data: %s\n\n", msg)
				if err := w.Flush(); err != nil {
					return
				}
			}
		}
	})

	return nil
}
