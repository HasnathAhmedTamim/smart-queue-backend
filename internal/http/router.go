package httpserver

import (
	"net/http"

	"github.com/hasnathahmedtamim/smart-queue/internal/http/handlers"
	"github.com/hasnathahmedtamim/smart-queue/internal/http/middleware"
)

func NewRouter(q *handlers.QueueHandler, allowedOrigin string) http.Handler {
	mux := http.NewServeMux()

	// REST
	mux.HandleFunc("POST /api/tokens", q.CreateToken)
	mux.HandleFunc("GET /api/queue", q.QueueStatus)
	mux.HandleFunc("POST /api/queue/next", q.Next)
	mux.HandleFunc("GET /api/services", q.ListServices)
	mux.HandleFunc("GET /api/tokens", q.ListTokens)

	// SSE
	mux.HandleFunc("GET /api/stream/queue", q.StreamQueue)

	// Middleware order: RequestID -> CORS
	h := middleware.RequestID(mux)
	h = middleware.CORS(allowedOrigin)(h)

	return h
}
