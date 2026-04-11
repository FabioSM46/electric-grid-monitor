package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"electric-grid-monitor/internal/collector"
	"electric-grid-monitor/internal/storage"
)

// Server handles HTTP API requests
type Server struct {
	store     *storage.Store
	collector *collector.Collector
	port      string
	host      string
}

// NewServer creates a new API server
func NewServer(store *storage.Store, collector *collector.Collector, host, port string) *Server {
	return &Server{
		store:     store,
		collector: collector,
		port:      port,
		host:      host,
	}
}

// Start begins the HTTP server. It blocks until the context is cancelled,
// then gracefully shuts down the server with a 5-second deadline.
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", s.handleHealth)

	// API v1 routes
	mux.HandleFunc("/api/v1/power/current", s.handleCurrentPower)
	mux.HandleFunc("/api/v1/history", s.handleHistory)
	mux.HandleFunc("/api/v1/stats/daily", s.handleDailyStats)

	addr := fmt.Sprintf("%s:%s", s.host, s.port)

	srv := &http.Server{
		Addr:    addr,
		Handler: s.corsMiddleware(mux),
	}

	// Shut down the server when the context is cancelled
	go func() {
		<-ctx.Done()
		log.Println("Shutting down API server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		}
	}()

	log.Printf("Starting API server on %s", addr)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}

// corsMiddleware adds CORS headers for Grafana
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}

func (s *Server) handleCurrentPower(w http.ResponseWriter, r *http.Request) {
	reading, err := s.collector.Collect()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to collect data: %v", err), http.StatusInternalServerError)
		return
	}

	// Also save to database
	if err := s.store.SaveReading(reading); err != nil {
		log.Printf("Warning: failed to save reading: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reading)
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	// Default to last 24 hours if not specified
	from := time.Now().Add(-24 * time.Hour)
	to := time.Now()

	if fromStr != "" {
		if parsed, err := time.Parse("2006-01-02", fromStr); err == nil {
			from = parsed
		}
	}

	if toStr != "" {
		if parsed, err := time.Parse("2006-01-02", toStr); err == nil {
			to = parsed.Add(24 * time.Hour) // End of the day
		}
	}

	readings, err := s.store.GetHistory(from, to)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get history: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"from":     from.Format(time.RFC3339),
		"to":       to.Format(time.RFC3339),
		"count":    len(readings),
		"readings": readings,
	})
}

func (s *Server) handleDailyStats(w http.ResponseWriter, r *http.Request) {
	dateStr := r.URL.Query().Get("date")

	date := time.Now()
	if dateStr != "" {
		if parsed, err := time.Parse("2006-01-02", dateStr); err == nil {
			date = parsed
		}
	}

	stats, err := s.store.GetDailyStats(date)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get stats: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
