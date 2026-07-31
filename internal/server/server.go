package server

import (
	"context"
	"encoding/json"
	"net/http"

	"carina/internal/platform/postgres"
)

type Server struct {
	db     *postgres.Pool
	router http.Handler
}

func New(cfg config.ServerConfig, db *postgres.Pool) *Server {
	s := &Server{
		db: db,
	}

	s.router = s.setupRouter()
	return s
}

func (s *Server) Handler() http.Handler {
	return s.router
}

func (s *Server) setupRouter() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", s.healthHandler)
	mux.HandleFunc("GET /health/ready", s.readyHandler)

	return mux
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{"status": "ok"}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) readyHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	if err := s.db.Health(ctx); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "unavailable",
			"details": "database connection failed",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
}