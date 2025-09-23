package ws

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type WSMessageType string

const (
	// System / notifications
	MsgTypeNotification  WSMessageType = "notification"
	MsgTypeRoadmapReady  WSMessageType = "roadmap_ready"
	MsgTypeQuizReady     WSMessageType = "quiz_ready"
	MsgTypeChatMessage   WSMessageType = "chat_message"
	MsgTypeLatencyReport WSMessageType = "latency_report"
	MsgTypeUnknown       WSMessageType = "unknown"
)

type WSMessage struct {
	Type       WSMessageType `json:"type"`
	Payload    interface{}   `json:"payload"`
	ReceiverID string        `json:"receiverID"`
	SenderID   string        `json:"senderId,omitempty"`
	Timestamp  time.Time     `json:"timestamp,omitempty"`
}

func (m *WSMessage) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

func FromJSON(data []byte) (*WSMessage, error) {
	var msg WSMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

func (m *WSMessage) SendMessageToUser(redis *redis.ClusterClient) bool {
	ctx := context.Background()

	// Get all servers the receiver is connected to
	serverIDs, err := redis.SMembers(ctx, "user_servers:"+m.ReceiverID).Result()
	if err != nil {
		log.Printf("failed to get user_servers for %s: %v", m.ReceiverID, err)
		return false
	}

	if len(serverIDs) == 0 {
		log.Printf("user %s is not connected to any server", m.ReceiverID)
		return false
	}

	payload, err := m.ToJSON()
	if err != nil {
		log.Printf("failed to marshal message: %v", err)
		return false
	}

	for _, serverID := range serverIDs {
		// Check if user is online on that server
		key := "user_online:" + m.ReceiverID + ":" + serverID
		exists, err := redis.Exists(ctx, key).Result()
		if err != nil {
			log.Printf("failed to check presence for %s on %s: %v", m.ReceiverID, serverID, err)
			continue
		}

		if exists == 0 {
			// stale entry, remove from user_servers
			if err := redis.SRem(ctx, "user_servers:"+m.ReceiverID, serverID).Err(); err != nil {
				log.Printf("failed to remove stale server mapping for %s: %v", m.ReceiverID, err)
			}
			continue
		}

		// Publish to server channel
		channel := "server:" + serverID
		if err := redis.Publish(ctx, channel, string(payload)).Err(); err != nil {
			log.Printf("failed to publish to %s: %v", channel, err)
		}
	}

	return true
}
