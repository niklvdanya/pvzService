package infra

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"gitlab.ozon.dev/safariproxd/homework/internal/workerpool"
)

type AdminServer struct {
	srv  *http.Server
	pool *workerpool.Pool
}

func NewAdmin(addr string, pool *workerpool.Pool) *AdminServer {
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
	}
	as := &AdminServer{srv: server, pool: pool}
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
	return as
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
