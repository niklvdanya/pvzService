package infra

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"gitlab.ozon.dev/safariproxd/homework/internal/workerpool"
)

type CacheManager interface {
	GetCacheStats() map[string]int
	ClearCache()
	CleanupExpired()
}

type AdminServer struct {
	srv          *http.Server
	pool         *workerpool.Pool
	cacheManager CacheManager
}

func NewAdmin(addr string, pool *workerpool.Pool, cacheManager CacheManager) *AdminServer {
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
	}
	as := &AdminServer{
		srv:          server,
		pool:         pool,
		cacheManager: cacheManager,
	}

	mux.HandleFunc("/resize", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "use POST", http.StatusMethodNotAllowed)
			return
		}
		n, _ := strconv.Atoi(r.URL.Query().Get("workers"))
		if n <= 0 {
			http.Error(w, "workers must be > 0", http.StatusBadRequest)
			return
		}
		as.pool.Resize(n)
		if _, err := w.Write([]byte("ok")); err != nil {
			slog.Warn("admin write failed", "error", err)
		}
	})

	mux.HandleFunc("/cache/stats", as.handleCacheStats)
	mux.HandleFunc("/cache/clear", as.handleCacheClear)
	mux.HandleFunc("/cache/cleanup", as.handleCacheCleanup)

	return as
}

func (a *AdminServer) handleCacheStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "use GET", http.StatusMethodNotAllowed)
		return
	}

	if a.cacheManager == nil {
		http.Error(w, "cache not available", http.StatusServiceUnavailable)
		return
	}

	stats := a.cacheManager.GetCacheStats()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		slog.Error("failed to encode cache stats", "error", err)
		http.Error(w, "encoding error", http.StatusInternalServerError)
	}
}

func (a *AdminServer) handleCacheClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "use POST", http.StatusMethodNotAllowed)
		return
	}

	if a.cacheManager == nil {
		http.Error(w, "cache not available", http.StatusServiceUnavailable)
		return
	}

	a.cacheManager.ClearCache()

	if _, err := w.Write([]byte("cache cleared")); err != nil {
		slog.Warn("admin write failed", "error", err)
	}
}

func (a *AdminServer) handleCacheCleanup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "use POST", http.StatusMethodNotAllowed)
		return
	}

	if a.cacheManager == nil {
		http.Error(w, "cache not available", http.StatusServiceUnavailable)
		return
	}

	a.cacheManager.CleanupExpired()

	if _, err := w.Write([]byte("expired cache entries cleaned")); err != nil {
		slog.Warn("admin write failed", "error", err)
	}
}

func (a *AdminServer) Start() {
	go func() {
		if err := a.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("admin listen error", "error", err)
		}
	}()
}

func (a *AdminServer) Shutdown(ctx context.Context) {
	if err := a.srv.Shutdown(ctx); err != nil {
		slog.Warn("admin shutdown error", "error", err)
	}
}
