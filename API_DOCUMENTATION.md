# MangaHub API Documentation

## Overview

MangaHub is a network-centric manga management system that implements multiple network protocols:
- **HTTP/REST API** - RESTful API with JWT authentication
- **TCP Socket** - JSON-based message protocol for real-time updates
- **UDP Broadcasting** - Notification system for client registration
- **gRPC** - Protocol Buffer-based service for efficient communication
- **WebSocket** - Real-time chat and messaging

## Server Ports

- **HTTP Server**: `:8080`
- **TCP Server**: `:8081`
- **UDP Server**: `:8082` (listening), `:8083` (broadcast)
- **gRPC Server**: `:8084`
- **WebSocket**: `/ws` (on HTTP server)

## HTTP REST API

### Base URL
```
http://localhost:8080/api/v1
```

### Authentication

All protected endpoints require a JWT token in the Authorization header:
```
Authorization: Bearer <token>
```

### Endpoints

#### Authentication

##### Register User
```http
POST /api/v1/auth/register
Content-Type: application/json

{
  "username": "string",
  "password": "string" (min 6 characters)
}
```

**Response:**
- `201 Created`: User registered successfully
  ```json
  {
    "message": "User registered successfully",
    "user_id": "uuid"
  }
  ```
- `400 Bad Request`: Invalid request body
- `409 Conflict`: Username already exists
- `500 Internal Server Error`: Server error

##### Login
```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "string",
  "password": "string"
}
```

**Response:**
- `200 OK`: Login successful
  ```json
  {
    "token": "jwt_token",
    "user_id": "uuid",
    "username": "string"
  }
  ```
- `400 Bad Request`: Invalid request body
- `401 Unauthorized`: Invalid credentials
- `500 Internal Server Error`: Server error

#### Manga

##### Get All Manga
```http
GET /api/v1/manga
```

**Response:**
- `200 OK`: List of manga
  ```json
  [
    {
      "id": "string",
      "title": "string",
      "author": "string",
      "genres": ["string"],
      "status": "string",
      "total_chapters": 0,
      "description": "string",
      "cover_url": "string"
    }
  ]
  ```

##### Get Manga by ID
```http
GET /api/v1/manga/:id
```

**Response:**
- `200 OK`: Manga details
- `404 Not Found`: Manga not found

##### Search Manga
```http
GET /api/v1/manga/search?q=query_string
```

**Response:**
- `200 OK`: List of matching manga
- `400 Bad Request`: Query parameter missing

##### Create Manga (Protected)
```http
POST /api/v1/manga
Authorization: Bearer <token>
Content-Type: application/json

{
  "id": "string",
  "title": "string",
  "author": "string",
  "genres": ["string"],
  "status": "string",
  "total_chapters": 0,
  "description": "string",
  "cover_url": "string"
}
```

**Response:**
- `201 Created`: Manga created (triggers UDP broadcast)
- `400 Bad Request`: Invalid request body
- `401 Unauthorized`: Missing or invalid token
- `500 Internal Server Error`: Server error

#### User Library (Protected)

##### Get User Library
```http
GET /api/v1/library
Authorization: Bearer <token>
```

**Response:**
- `200 OK`: List of manga in user's library
  ```json
  [
    {
      "id": "string",
      "user_id": "string",
      "manga_id": "string",
      "status": "reading|completed|plan_to_read|dropped",
      "added_at": "timestamp"
    }
  ]
  ```

##### Add to Library
```http
POST /api/v1/library
Authorization: Bearer <token>
Content-Type: application/json

{
  "manga_id": "string",
  "status": "string" (optional, default: "plan_to_read")
}
```

**Response:**
- `201 Created`: Manga added to library
- `400 Bad Request`: Invalid request body
- `401 Unauthorized`: Missing or invalid token

##### Update Library Status
```http
PUT /api/v1/library/:id
Authorization: Bearer <token>
Content-Type: application/json

{
  "status": "reading|completed|plan_to_read|dropped"
}
```

**Response:**
- `200 OK`: Status updated
- `400 Bad Request`: Invalid request body
- `401 Unauthorized`: Missing or invalid token

##### Remove from Library
```http
DELETE /api/v1/library/:id
Authorization: Bearer <token>
```

**Response:**
- `200 OK`: Removed from library
- `401 Unauthorized`: Missing or invalid token

#### Progress (Protected)

##### Get User Progress
```http
GET /api/v1/progress
Authorization: Bearer <token>
```

**Response:**
- `200 OK`: List of all progress entries
  ```json
  [
    {
      "id": "string",
      "user_id": "string",
      "manga_id": "string",
      "chapter": 0,
      "updated_at": "timestamp"
    }
  ]
  ```

##### Get Manga Progress
```http
GET /api/v1/progress/:id
Authorization: Bearer <token>
```

**Response:**
- `200 OK`: Progress for specific manga
- `404 Not Found`: Progress not found

##### Update Progress
```http
POST /api/v1/progress
Authorization: Bearer <token>
Content-Type: application/json

{
  "manga_id": "string",
  "chapter": 0
}
```

**Response:**
- `200 OK`: Progress updated (triggers TCP and UDP broadcasts)
  ```json
  {
    "id": "string",
    "user_id": "string",
    "manga_id": "string",
    "chapter": 0,
    "updated_at": "timestamp"
  }
  ```
- `400 Bad Request`: Invalid request body
- `401 Unauthorized`: Missing or invalid token

### HTTP Status Codes

- `200 OK`: Request successful
- `201 Created`: Resource created successfully
- `400 Bad Request`: Invalid request parameters
- `401 Unauthorized`: Authentication required or failed
- `404 Not Found`: Resource not found
- `409 Conflict`: Resource conflict (e.g., duplicate username)
- `500 Internal Server Error`: Server error

### CORS

The API supports CORS for web clients:
- `Access-Control-Allow-Origin: *`
- `Access-Control-Allow-Methods: POST, GET, PUT, DELETE, PATCH, OPTIONS`
- `Access-Control-Allow-Headers: Content-Type, Authorization`

## TCP Socket Protocol

### Connection
Connect to `localhost:8081` using TCP.

### Message Format
All messages are JSON-encoded and newline-delimited:
```json
{
  "type": "string",
  "user_id": "string",
  "manga_id": "string",
  "chapter": 0,
  "data": {},
  "timestamp": "RFC3339"
}
```

### Message Types

#### Register
```json
{
  "type": "register",
  "user_id": "string"
}
```

**Response:**
```json
{
  "type": "registered",
  "user_id": "string",
  "timestamp": "RFC3339"
}
```

#### Progress Update
```json
{
  "type": "progress_update",
  "user_id": "string",
  "manga_id": "string",
  "chapter": 0
}
```

**Response:**
```json
{
  "type": "progress_ack",
  "timestamp": "RFC3339"
}
```

The server broadcasts this update to all other connected clients.

#### Ping
```json
{
  "type": "ping"
}
```

**Response:**
```json
{
  "type": "pong",
  "timestamp": "RFC3339"
}
```

#### Progress Broadcast (Server → Clients)
When progress is updated via HTTP API, all TCP clients receive:
```json
{
  "type": "progress_broadcast",
  "user_id": "string",
  "manga_id": "string",
  "chapter": 0,
  "timestamp": "RFC3339"
}
```

### Error Handling
- Connection timeout: 60 seconds
- Graceful disconnection on network errors
- Automatic client cleanup on failures

## UDP Notification Protocol

### Connection
Connect to `localhost:8082` using UDP.

### Message Format
JSON-encoded UDP datagrams:
```json
{
  "type": "string",
  "user_id": "string"
}
```

### Message Types

#### Register
```json
{
  "type": "register",
  "user_id": "string"
}
```

**Response:**
```json
{
  "type": "registered",
  "message": "Successfully registered for notifications",
  "timestamp": "RFC3339"
}
```

#### Heartbeat
```json
{
  "type": "heartbeat"
}
```

Clients should send heartbeat every 30 seconds to remain registered.

### Notification Types

#### New Manga
```json
{
  "type": "new_manga",
  "message": "New manga added: <title>",
  "data": {
    "manga_id": "string",
    "title": "string"
  },
  "timestamp": "RFC3339"
}
```

#### Progress Update
```json
{
  "type": "update",
  "message": "Progress updated",
  "data": {
    "user_id": "string",
    "manga_id": "string",
    "chapter": 0
  },
  "timestamp": "RFC3339"
}
```

### Client Management
- Clients are automatically removed after 2 minutes of inactivity
- Network failures result in automatic client removal
- Broadcast failures are logged and handled gracefully

## WebSocket Protocol

### Connection
```
ws://localhost:8080/ws
```

### Message Format
```json
{
  "type": "string",
  "user_id": "string",
  "username": "string",
  "content": "string",
  "data": {},
  "timestamp": "RFC3339"
}
```

### Message Types

#### Register
```json
{
  "type": "register",
  "user_id": "string",
  "username": "string"
}
```

**Response:**
```json
{
  "type": "registered",
  "user_id": "string",
  "username": "string",
  "timestamp": "RFC3339"
}
```

#### Chat
```json
{
  "type": "chat",
  "content": "message text"
}
```

Broadcasts to all connected clients:
```json
{
  "type": "chat",
  "user_id": "string",
  "username": "string",
  "content": "message text",
  "timestamp": "RFC3339"
}
```

#### Ping
```json
{
  "type": "ping"
}
```

**Response:**
```json
{
  "type": "pong",
  "timestamp": "RFC3339"
}
```

#### User Events
- `user_joined`: Broadcasted when a user connects
- `user_left`: Broadcasted when a user disconnects

### Connection Management
- Ping interval: 54 seconds
- Read timeout: 60 seconds
- Write timeout: 10 seconds
- Automatic cleanup of disconnected clients

## gRPC Service

### Service Definition
See `api/mangahub.proto` for complete Protocol Buffer definitions.

### Methods

#### GetManga
```protobuf
rpc GetManga(GetMangaRequest) returns (MangaResponse);
```

#### ListManga
```protobuf
rpc ListManga(ListMangaRequest) returns (ListMangaResponse);
```

#### SearchManga
```protobuf
rpc SearchManga(SearchMangaRequest) returns (ListMangaResponse);
```

#### GetUserProgress
```protobuf
rpc GetUserProgress(GetUserProgressRequest) returns (UserProgressResponse);
```

#### UpdateProgress
```protobuf
rpc UpdateProgress(UpdateProgressRequest) returns (UpdateProgressResponse);
```

### Connection
Connect to `localhost:8084` using gRPC.

### Error Handling
- Uses gRPC status codes (codes.InvalidArgument, codes.NotFound, codes.Internal)
- Proper error messages for debugging

## Error Handling

All protocols implement comprehensive error handling:

1. **Network Errors**: Timeouts, connection failures, and network interruptions are detected and handled
2. **Protocol Errors**: Invalid messages, malformed JSON, and unknown message types are handled gracefully
3. **Client Cleanup**: Failed clients are automatically removed from connection pools
4. **Logging**: All errors are logged with appropriate context for debugging

## Performance Characteristics

- **Concurrent Connections**: Supports 50-100 concurrent users
- **TCP Connections**: Handles 20-30 concurrent TCP connections
- **WebSocket**: Supports 10-20 simultaneous chat users
- **Search Performance**: Basic queries complete within 500ms
- **Database**: SQLite with 30-40 manga series

## Security Considerations

1. **JWT Authentication**: All protected endpoints require valid JWT tokens
2. **Password Hashing**: Uses bcrypt with cost factor 12
3. **CORS**: Configured for web client access
4. **Input Validation**: Request validation on all endpoints
5. **SQL Injection**: Parameterized queries prevent SQL injection

## Development Notes

- All servers support graceful shutdown
- Comprehensive logging for debugging
- Protocol integration: HTTP updates trigger TCP/UDP broadcasts
- Error recovery and client cleanup mechanisms
- Connection lifecycle management for all protocols




