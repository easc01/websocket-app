## 🚀 Redis Usage

### 1. **Caching**

* Store frequently accessed data (e.g. roadmap metadata, user preferences, or lightweight lookup tables).
    
* Example:
    
    ```
    roadmap:{roadmapId} → JSON roadmap data
    ```
    

* * *

### 2. **Presence & Online Tracking**

* Keys that represent if a user is currently connected to a WebSocket server.
    
* Mechanism: heartbeat every 30s, TTL ~60s.
    
* Example:
    
    ```
    user_online:{userId}:{serverId} → "1" (expires in 60s)
    ```
    

* * *

### 3. **User-to-Server Mapping**

* Keep track of which WebSocket servers a user is connected to.
    
* Example:
    
    ```
    user_servers:{userId} → {serverId1, serverId2}
    ```
    
* Used when you need to deliver a message → check which server(s) hold the connection.
    

* * *

### 4. **Inter-Server Communication**

* Each WebSocket server has its own pub/sub channel.
    
* Example:
    
    ```
    server:{serverId}
    ```
    
* Worker or API publishes an event → only the server hosting the user consumes and pushes via WebSocket.
    

* * *

### 5. **(P2) Group Messaging**

* Each group chat has its own pub/sub channel for broadcasting.
    
* Example:
    
    ```
    group:{groupId}
    ```
    
* Servers subscribed to this channel deliver messages to all connected group members locally.
    

* * *

✅ **In short**:

* **Cache** = fast lookups
    
* **Presence** = `user_online` with TTL
    
* **Routing** = `user_servers` set
    
* **Message bus** = `server:{serverId}` channel
    
* **Groups (P2)** = `group:{groupId}` channel
    

* * *
