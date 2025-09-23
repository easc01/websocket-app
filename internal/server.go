package ws

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/easc01/websocket-app/pkg/metrics"
	"github.com/easc01/websocket-app/pkg/ws"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

type Server struct {
	serverID string
	redis    *redis.ClusterClient
	upgrader websocket.Upgrader
	clients  map[string]map[string]*Client // userId -> connectionId -> Client
	mu       sync.RWMutex
}

func NewServer(serverID string, redisClient *redis.ClusterClient) *Server {
	return &Server{
		serverID: serverID,
		redis:    redisClient,
		upgrader: websocket.Upgrader{
			CheckOrigin:     func(r *http.Request) bool { return true },
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		clients: make(map[string]map[string]*Client),
	}
}

// Handle incoming websocket connections
func (s *Server) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("userId")
	if userID == "" {
		http.Error(w, "missing userId", http.StatusBadRequest)
		return
	}

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrade failed: %v", err)
		return
	}

	client, err := s.AddClient(userID, conn)
	if err != nil {
		log.Printf("failed to register user %s: %v", userID, err)
		conn.Close()
		return
	}

	// listen for ws messages for client, once disconnected, unregister it
	go func(c *Client) {
		c.Listen()
		s.RemoveClient(c)
	}(client)
}

func (s *Server) AddClient(userID string, conn *websocket.Conn) (*Client, error) {
	ctx := context.Background()

	c := &Client{
		ID:       uuid.New().String(),
		UserID:   userID,
		ServerID: s.serverID,
		Conn:     conn,
		Redis:    s.redis,
	}

	s.mu.Lock()
	if s.clients[userID] == nil {
		s.clients[userID] = make(map[string]*Client)
	}
	s.clients[userID][c.ID] = c
	s.mu.Unlock()

	// add server to user_servers
	if err := s.redis.SAdd(ctx, "user_servers:"+c.UserID, c.ServerID).Err(); err != nil {
		return nil, err
	}

	// mark user online
	if err := s.redis.Set(ctx, "user_online:"+c.UserID+":"+c.ServerID, "1", 60*time.Second).Err(); err != nil {
		return nil, err
	}

	metrics.OnClientConnect()
	log.Printf("user %s connected to server %s", userID, s.serverID)
	return c, nil
}

func (s *Server) RemoveClient(c *Client) {
	ctx := context.Background()
	if c.cancel != nil {
		c.cancel() // stop heartbeat
	}

	s.mu.Lock()
	if userClients, ok := s.clients[c.UserID]; ok {
		delete(userClients, c.ID)
		if len(userClients) == 0 {
			delete(s.clients, c.UserID)
		}
	}
	s.mu.Unlock()

	_ = c.Conn.Close()

	// remove server mapping
	if err := c.Redis.SRem(ctx, "user_servers:"+c.UserID, c.ServerID).Err(); err != nil {
		log.Printf("failed to remove server mapping: %v", err)
	}

	// remove presence key
	if err := c.Redis.Del(ctx, "user_online:"+c.UserID+":"+c.ServerID).Err(); err != nil {
		log.Printf("failed to delete presence key: %v", err)
	}

	metrics.OnClientDisconnect()
	log.Printf("user %s disconnected from server %s", c.UserID, s.serverID)
}

func (s *Server) ConsumeServerEvents(ctx context.Context) {
	// Subscribe to this server's channel
	serverChannelName := "server:" + s.serverID
	sub := s.redis.Subscribe(ctx, serverChannelName)

	// Make sure to close the subscription when the function exits
	go func() {
		ch := sub.Channel()
		log.Printf("Subscribed to Redis channel: %s", serverChannelName)

		for {
			select {
			case <-ctx.Done():
				// App is shutting down, unsubscribe and exit
				if err := sub.Close(); err != nil {
					log.Printf("failed to close Redis subscription: %v", err)
				} else {
					log.Printf("unsubscribed from Redis channel: %s", serverChannelName)
				}
				return
			case msg, ok := <-ch:
				if !ok {
					// Channel closed
					return
				}

				log.Printf("Received message from Redis channel %s: %s", msg.Channel, msg.Payload)

				// Parse the JSON into WSMessage
				wsMsg, err := ws.FromJSON([]byte(msg.Payload))
				if err != nil {
					log.Printf("invalid message from redis pubsub: %v", err)
					continue
				}

				// Send only to the intended user (local server)
				s.SendMessageToLocalUser(wsMsg)
			}
		}
	}()
}

// SendMessageToLocalUser sends a message to all active connections of a specific user
func (s *Server) SendMessageToLocalUser(message *ws.WSMessage) {
	if message.ReceiverID == "" {
		log.Printf("receiverID is empty, cannot send message: %+v", message)
		return
	}

	payload, err := message.ToJSON()
	if err != nil {
		log.Printf("failed to marshal message: %v", err)
		return
	}

	s.mu.RLock()
	userClients, ok := s.clients[message.ReceiverID]
	s.mu.RUnlock()

	if !ok {
		// user not connected to this server
		return
	}

	for _, client := range userClients {
		if err := client.Conn.WriteMessage(websocket.TextMessage, payload); err != nil {
			log.Printf("failed to send message to user %s: %v", client.UserID, err)
		} else {
			metrics.OnMessageDelivered()
		}
	}
}
