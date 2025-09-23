package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/easc01/websocket-app/internal"
	"github.com/easc01/websocket-app/pkg/config"
	"github.com/easc01/websocket-app/pkg/metrics"
	"github.com/easc01/websocket-app/pkg/redis"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// load config
	cfg := config.AppConfig

	// connect redis
	redis.Connect()
	defer redis.Close()

	// push metrics to CW every minute
	metrics.StartCloudWatchPusher(time.Second * 60)

	// start websocket server
	wsServer := ws.NewServer(cfg.ServerID, redis.Client)
	wsServer.ConsumeServerEvents(ctx)

	http.HandleFunc("/ws", wsServer.HandleWebSocket)

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	log.Printf("WebSocket server [%s] listening on :%s\n", cfg.ServerID, cfg.WSSPort)
	if err := http.ListenAndServe(":"+cfg.WSSPort, nil); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
