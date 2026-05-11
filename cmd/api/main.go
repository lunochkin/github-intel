package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lunochkin/github-intel/internal/clickhouse"
	"github.com/lunochkin/github-intel/internal/dotenv"
	"github.com/lunochkin/github-intel/internal/server"
)

func main() {
	if err := dotenv.Load(); err != nil {
		log.Fatalf("load .env: %v", err)
	}

	ctxBG := context.Background()
	chConn, err := clickhouse.Open(ctxBG)
	if err != nil {
		log.Fatalf("clickhouse: %v", err)
	}
	defer chConn.Close()

	addr := os.Getenv("LISTEN_ADDR")
	srv := &http.Server{
		Addr:         addr,
		Handler:      server.NewMux(chConn),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		ctx, cancel := server.ShutdownTimeout()
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("shutdown: %v", err)
		}
	}()

	log.Printf("api listening on http://localhost%s/summary", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
