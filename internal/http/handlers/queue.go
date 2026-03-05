package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/hasnathahmedtamim/smart-queue/internal/realtime"
	"github.com/hasnathahmedtamim/smart-queue/internal/service"
	"github.com/hasnathahmedtamim/smart-queue/internal/types"
	"github.com/hasnathahmedtamim/smart-queue/internal/utils/response"
)

type QueueHandler struct {
	svc      *service.QueueService
	validate *validator.Validate
	adminKey string
	hub      *realtime.Hub
}

func NewQueueHandler(svc *service.QueueService, adminKey string, hub *realtime.Hub) *QueueHandler {
	return &QueueHandler{
		svc:      svc,
		validate: validator.New(),
		adminKey: adminKey,
		hub:      hub,
	}
}

// POST /api/tokens
func (h *QueueHandler) CreateToken(w http.ResponseWriter, r *http.Request) {
	var req types.CreateTokenRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "BAD_JSON", "invalid JSON body")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	token, pos, est, err := h.svc.CreateToken(r.Context(), req.ServiceCode, strings.TrimSpace(req.CustomerName))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "CREATE_FAILED", err.Error())
		return
	}

	// Broadcast queue update (so frontend updates instantly)
	h.broadcastQueueUpdate(r)

	response.JSON(w, http.StatusCreated, types.CreateTokenResponse{
		Token:         token,
		Position:      pos,
		EstimatedMins: est,
	})
}

// GET /api/queue
func (h *QueueHandler) QueueStatus(w http.ResponseWriter, r *http.Request) {
	cur, waiting, err := h.svc.QueueStatus(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "SERVER_ERROR", err.Error())
		return
	}
	response.JSON(w, http.StatusOK, types.QueueStatusResponse{
		CurrentToken: cur,
		Waiting:      waiting,
	})
}

// POST /api/queue/next  (admin)
func (h *QueueHandler) Next(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Admin-Key") != h.adminKey {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid admin key")
		return
	}

	cur, err := h.svc.Next(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "SERVER_ERROR", err.Error())
		return
	}

	// Broadcast queue update
	h.broadcastQueueUpdate(r)

	response.JSON(w, http.StatusOK, types.NextResponse{
		CurrentToken: cur,
	})
}

// GET /api/stream/queue  (SSE)
func (h *QueueHandler) StreamQueue(w http.ResponseWriter, r *http.Request) {
	// SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Subscribe client to hub
	ch := h.hub.Subscribe()
	defer h.hub.Unsubscribe(ch)

	// Send initial snapshot
	cur, waiting, _ := h.svc.QueueStatus(r.Context())
	initPayload := []byte(fmt.Sprintf(`{"type":"snapshot","current_token":%q,"waiting":%d}`, cur, waiting))
	writeSSE(w, "queue", initPayload)
	flusher.Flush()

	// Keepalive ping
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case msg := <-ch:
			writeSSE(w, "queue", msg)
			flusher.Flush()
		case <-ticker.C:
			_, _ = w.Write([]byte(": ping\n\n"))
			flusher.Flush()
		}
	}
}

func (h *QueueHandler) broadcastQueueUpdate(r *http.Request) {
	cur, waiting, _ := h.svc.QueueStatus(r.Context())
	payload := []byte(fmt.Sprintf(`{"type":"update","current_token":%q,"waiting":%d}`, cur, waiting))
	h.hub.Publish(payload)
}

// GET /api/services
func (h *QueueHandler) ListServices(w http.ResponseWriter, r *http.Request) {
	services, err := h.svc.ListServices(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "SERVER_ERROR", err.Error())
		return
	}

	// Convert to JSON-friendly slice
	out := make([]map[string]string, 0, len(services))
	for _, s := range services {
		out = append(out, map[string]string{
			"code": s.Code,
			"name": s.Name,
		})
	}

	response.JSON(w, http.StatusOK, out)
}

// GET /api/tokens?status=waiting|serving|done&limit=50
func (h *QueueHandler) ListTokens(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "waiting"
	}

	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}

	items, err := h.svc.ListTokensByStatus(r.Context(), status, limit)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	response.JSON(w, http.StatusOK, items)
}

func writeSSE(w http.ResponseWriter, event string, data []byte) {
	_, _ = w.Write([]byte("event: " + event + "\n"))
	_, _ = w.Write([]byte("data: "))
	_, _ = w.Write(data)
	_, _ = w.Write([]byte("\n\n"))
}
