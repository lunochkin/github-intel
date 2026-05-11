package server

import (
	"context"
	"net/http"
	"time"

	driverch "github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/lunochkin/github-intel/internal/summary"
)

func NewMux(conn driverch.Conn) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("GET /summary", summary.Handler(conn))
	return mux
}

func ShutdownTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 10*time.Second)
}
