package httpserver

import (
	"net/http"

	"github.com/hasnathahmedtamim/smart-queue/internal/http/handlers"
	"github.com/hasnathahmedtamim/smart-queue/internal/http/middleware"
)

func NewRouter(q *handlers.QueueHandler) http.Handler {
	mux := http.NewServeMux()

	// REST
	mux.HandleFunc("POST /api/tokens", q.CreateToken)
	mux.HandleFunc("GET /api/queue", q.QueueStatus)
	mux.HandleFunc("POST /api/queue/next", q.Next)

	// SSE
	mux.HandleFunc("GET /api/stream/queue", q.StreamQueue)

	// Middleware order: RequestID -> CORS
	h := middleware.RequestID(mux)
	h = middleware.CORS("http://localhost:3000")(h)

	return h
}
