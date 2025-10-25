package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"order/internal/app"
	"order/internal/bus"
	"order/internal/cache"
	"order/internal/config"
	"order/internal/httpapi"
	"order/internal/storage"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	store, err := storage.NewPostgres(ctx, cfg.PGURL)
	must(err)
	defer store.Close()

	mem := cache.NewMem()

	sub, err := bus.NewStan(cfg.StanCluster, cfg.StanClient, cfg.StanURL)
	must(err)
	defer sub.Close()

	application := &app.App{Store: store, Cache: mem, Sub: sub}
	must(application.RestoreCache(ctx))

	// подписка
	must(sub.Start(cfg.StanSubject, application.HandleMsg))

	// http
	srv := httpapi.New(mem)
	go func() { must(srv.Listen(cfg.HTTPAddr)) }()

	// graceful shutdown
	waitForCtrlC()
	log.Println("bye")
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func waitForCtrlC() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	<-ch
}
