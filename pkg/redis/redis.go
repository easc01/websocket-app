package redis

import (
	"context"
	"log"
	"time"

	"net/url"

	"github.com/easc01/websocket-app/pkg/config"
	"github.com/redis/go-redis/v9"
)

var Client *redis.ClusterClient

func Connect() {
	redisURI := config.AppConfig.RedisConnectionURI
	if redisURI == "" {
		log.Fatal("REDIS_URL not set")
	}

	// Parse the URI
	u, err := url.Parse(redisURI)
	if err != nil {
		log.Fatalf("Invalid REDIS_URL: %v", err)
	}

	password := ""
	if u.User != nil {
		password, _ = u.User.Password()
	}

	// Use cluster client for cluster mode
	opts := &redis.ClusterOptions{
		Addrs:    []string{u.Host}, // configuration endpoint
		Password: password,
		PoolSize:     10,
		MinIdleConns: 2,
		PoolTimeout:  30 * time.Second,
	}

	Client = redis.NewClusterClient(opts)

	// Ping with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := Client.Ping(ctx).Err(); err != nil {
		log.Fatalf("failed to connect to Redis cluster: %v", err)
	}

	log.Println("Redis cluster connected successfully over TLS")
}

func Close() {
	if Client != nil {
		_ = Client.Close()
	}
}
