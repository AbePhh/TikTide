package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AbePhh/TikTide/backend/internal/app"
	httprouter "github.com/AbePhh/TikTide/backend/internal/http/router"
	"github.com/AbePhh/TikTide/backend/pkg/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	appCtx, err := app.New(cfg)
	if err != nil {
		log.Fatalf("build app context: %v", err)
	}
	defer func() {
		if err := appCtx.Close(); err != nil {
			log.Printf("close resources: %v", err)
		}
	}()

	engine := httprouter.NewEngine(appCtx)
	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           engine,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("%s server listening on %s", cfg.AppName, cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server failed: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("shutdown http server: %v", err)
	}
}
