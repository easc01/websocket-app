# websocket-app


`websocket-app` is a high-performance, horizontally scalable WebSocket server framework built in Go, designed to handle real-time messaging at large scale. It is intended for applications such as chat platforms, live dashboards, multiplayer games, and any system requiring real-time communication.

The architecture emphasizes **non-sticky connections**, enabling users to connect to any server instance without requiring session affinity, while **Redis pub/sub** ensures message fan-out across multiple servers. This approach allows the system to scale horizontally seamlessly, as additional server instances can be added to handle increased client load.

The project is fully instrumented with **CloudWatch metrics**, allowing monitoring of message throughput, connection counts, latency, and unexpected disconnects, providing visibility into system performance under heavy load.


## Architecture

![architecture](/docs/assets/image.png)

The architecture of `websocket-app` is designed around **high availability, horizontal scaling, and minimal latency**:

1. **Client Layer**: Web clients connect via WebSocket, sending and receiving messages in real-time. Each client is assigned a unique connection ID.
    
2. **WebSocket Servers**: Multiple EC2 instances run the Go-based server. Servers maintain a **local map of active clients**, keyed by user ID and connection ID, and handle WebSocket upgrade, message parsing, and client heartbeats.
    
3. **Redis Cluster**: Redis acts as the coordination layer. It stores:
    
    * `user_servers:<userID>`: The set of servers a user is connected to, acts as a route table here.
        
    * `server:<serverID>`: Pub/sub channel for fan-out messages to users connected to that server
        
4. **Message Flow**:
    
    * Incoming messages are handled by the local server.
        
    * For broadcast or messages to users on other servers, the message is published to Redis channels.
        
    * Each server subscribes to its own Redis channel and delivers messages to connected clients.
        
5. **Metrics Layer**: Metrics such as total messages, delivered messages, unexpected disconnects, active connections, and average latency are tracked and pushed to **AWS CloudWatch** every 60 seconds.
    

```
Clients → WebSocket Servers → Redis Cluster (pub/sub) → Other WS Servers → Receiver Clients
```


## Installation & Setup

### Prerequisites

* [Go 1.24.2](https://go.dev/)
    
* [Redis](https://redis.io/) (Cluster enabled)
    
* AWS Access Id and Secret Key in environment (aws cli configure recommended)
    
* Environment variables for configuration
    

### Environment Variables

Create a `.env.local` file with the following variables mentioned in [.env.example](./.env.example).



### Running Locally

```bash
go mod tidy
go run cmd/main.go
```

## Client Lifecycle

1. **Connection**: When a client connects, it is upgraded to a WebSocket, assigned a unique ID, and registered both in the local server and Redis.
    
2. **Heartbeat**: Each client sends a heartbeat every 30 seconds to keep connection alive.
    
3. **Message Handling**:
    
    * Chat messages are published to Redis channels corresponding to the servers where the receiver is connected.
        
    * Latency reports update CloudWatch metrics. (Optional, just for metrics)
        
4. **Disconnection**: When a client disconnects, all associated keys and mappings are removed, and metrics are updated.
    


## Redis Key Structure

| Key Pattern | Purpose |
| --- | --- |
| `user_servers:<userID>` | Tracks which servers a user is connected to |
| `server:<serverID>` | Pub/sub channel for message fan-out |


## Metrics

The system pushes the following metrics to AWS CloudWatch:

* **MessagesTotal**: Total messages processed per server
* **MessagesDelivered**: Successfully delivered messages
* **ActiveConnections**: Number of live client connections
* **AverageLatencyMs**: Average message delivery latency
    

Metrics are published every **60 seconds** and can be visualized on CloudWatch dashboards.


## Scaling & Performance

* **Horizontal Scaling**: Non-sticky connections allow any user to connect to any server instance. Additional EC2 instances can be added linearly to support more concurrent clients.
    
* **Redis Cluster**: A high-throughput Redis cluster ensures fast pub/sub operations for message fan-out.
    
* **Resource Usage**:
    
    * For **10k clients**, 4 × c6i.large instances and a Xlarge Redis cluster are sufficient.
        
    * For **30k clients**, consider scaling linearly (e.g., 8 × c6i.large) while monitoring metrics.
        
* **Optimization Notes**:
    
    * Messages are only sent to servers where the user is connected.
        
    * Let Load Balancer terminate the connection, and on termination, remove stale redis keys.
        
    * Pub/sub avoids server-to-server direct message passing, reducing inter-server network load.
        


## Running Stress Tests

* Use automated WebSocket clients to simulate load.
    
* Monitor CloudWatch metrics for latency, message throughput.
    
* Adjust instance counts or Redis cluster size if metrics exceed thresholds.
  
* Use the client side code at this [repo](https://github.com/easc01/ws-load-test) to simulate automated ws clients.