package ws

import (
	"context"
	"log"
	"time"

	"github.com/easc01/websocket-app/pkg/metrics"
	"github.com/easc01/websocket-app/pkg/ws"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

type Client struct {
	ID       string
	UserID   string
	ServerID string
	Conn     *websocket.Conn
	Redis    *redis.ClusterClient
	cancel   context.CancelFunc
}

// Keep refreshing user_online TTL
func (c *Client) Heartbeat(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("stopping heartbeat for user %s", c.UserID)
			return
		case <-ticker.C:
			err := c.Redis.Expire(ctx, "user_online:"+c.UserID+":"+c.ServerID, 60*time.Second).Err()
			if err != nil {
				log.Printf("heartbeat refresh failed for %s: %v", c.UserID, err)
			}
		}
	}
}

func (c *Client) Listen() {
	// create a cancellable context for heartbeat
	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel

	// start heartbeat in a goroutine
	go c.Heartbeat(ctx)

	for {
		_, msgBytes, err := c.Conn.ReadMessage()
		if err != nil {
			// Check if it’s a clean close
			if closeErr, ok := err.(*websocket.CloseError); ok {
				if closeErr.Code == websocket.CloseNormalClosure ||
					closeErr.Code == websocket.CloseGoingAway {
					log.Printf("connection closed normally for user %s: %v", c.UserID, err)
				} else {
					// unexpected close code
					metrics.OnUnexpectedDisconnect()
					log.Printf("unexpected disconnect for user %s: %v", c.UserID, err)
				}
			} else {
				// not a CloseError (network error, reset, etc.) → unexpected
				metrics.OnUnexpectedDisconnect()
				log.Printf("unexpected disconnect for user %s: %v", c.UserID, err)
			}
			break
		}

		// parse JSON into WSMessage
		msg, err := ws.FromJSON(msgBytes)
		if err != nil {
			log.Printf("invalid message from user %s: %v", c.UserID, err)
			continue
		}

		msg.SenderID = c.UserID
		msg.Timestamp = time.Now()

		// handle the message
		metrics.OnMessageReceived()
		c.handleUserSentMessage(msg)
	}
}

func (c *Client) handleUserSentMessage(msg *ws.WSMessage) {
	switch msg.Type {
	case ws.MsgTypeChatMessage:
		// handle chat message
		log.Printf("recieved chat message from %s: %v", c.UserID, msg.Payload)
		msg.SendMessageToUser(c.Redis)

	default:
		log.Printf("unknown message type from %s: %v", c.UserID, msg.Type)
	}
}
