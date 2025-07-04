package infra

import (
	"context"
	"net/http"
	"strconv"

	"gitlab.ozon.dev/safariproxd/homework/internal/workerpool"
)

type AdminServer struct {
	srv  *http.Server
	pool *workerpool.Pool
}

func NewAdmin(addr string, pool *workerpool.Pool) *AdminServer {
	mux := http.NewServeMux()
	as := &AdminServer{srv: &http.Server{Addr: addr, Handler: mux}, pool: pool}

	mux.HandleFunc("/resize", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "use POST", http.StatusMethodNotAllowed)
			return
		}
		n, _ := strconv.Atoi(r.URL.Query().Get("workers"))
		if n <= 0 {
			http.Error(w, "workers must be > 0", 400)
			return
		}
		as.pool.Resize(n)
		w.Write([]byte("ok"))
	})
	return as
}

func (a *AdminServer) Start() {
	go a.srv.ListenAndServe()
}
func (a *AdminServer) Shutdown(ctx context.Context) {
	a.srv.Shutdown(ctx)
}
