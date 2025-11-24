# MangaHub - Network-Centric Manga Management System

A comprehensive manga management platform implementing multiple network communication protocols for a network-centric computing course.

## Overview

MangaHub is a full-featured manga library management system that demonstrates implementation of various network protocols:
- **HTTP/REST API** with JWT authentication
- **TCP Socket** communication with JSON protocol
- **UDP Broadcasting** for notifications
- **gRPC** services with Protocol Buffers
- **WebSocket** for real-time chat

## Features

### Network Protocols

1. **HTTP Services**
   - RESTful API with proper HTTP methods (GET, POST, PUT, DELETE)
   - JWT-based authentication middleware
   - Comprehensive error handling with appropriate HTTP status codes
   - CORS support for web clients
   - Graceful error responses

2. **TCP Socket Communication**
   - Server accepting multiple concurrent connections
   - JSON-based message protocol
   - Concurrent connection handling with goroutines
   - Graceful connection termination
   - Progress update broadcasting
   - Connection timeout and error recovery

3. **UDP Broadcasting**
   - UDP server for notifications
   - Client registration mechanism
   - Broadcast functionality with error handling
   - Automatic client cleanup for inactive connections
   - Network failure recovery

4. **gRPC Services**
   - Protocol Buffer message definitions
   - Complete service implementation
   - Client-server communication
   - Proper error handling with gRPC status codes

5. **WebSocket Connections**
   - WebSocket upgrade handling
   - Real-time message broadcasting
   - Connection lifecycle management
   - Client management with automatic cleanup
   - Ping/pong keepalive mechanism

### Core Functionality

- User registration and authentication
- Manga catalog management
- User library management (reading, completed, plan to read, dropped)
- Reading progress tracking
- Real-time chat via WebSocket
- Cross-protocol integration (HTTP updates trigger TCP/UDP broadcasts)

## Architecture

```
mangahub/
├── cmd/
│   └── api-server/        # Main application entry point
├── internal/
│   ├── auth/              # JWT authentication
│   ├── grpc/              # gRPC service implementation
│   ├── library/           # User library handlers
│   ├── manga/             # Manga handlers and data loading
│   ├── middleware/        # HTTP middleware (CORS)
│   ├── progress/          # Progress tracking handlers
│   ├── tcp/               # TCP server implementation
│   ├── udp/               # UDP server implementation
│   ├── user/              # User handlers
│   └── websocket/         # WebSocket hub and handlers
├── pkg/
│   ├── database/          # Database connection and schema
│   └── models/            # Data models
├── api/                   # Protocol Buffer definitions
└── data/                  # Initial manga data (JSON)
```

## Getting Started

### Prerequisites

- Go 1.25.1 or later
- Protocol Buffer compiler (protoc)
- Go plugins for protoc

### Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd mangahub
```

2. Install dependencies:
```bash
go mod download
```

3. Generate Protocol Buffer code (if needed):
```bash
# Windows
scripts\generate-proto.bat

# Linux/Mac
scripts/generate-proto.sh
```

4. Run the server:
```bash
go run cmd/api-server/main.go
```

### Server Ports

- **HTTP Server**: `:8080`
- **TCP Server**: `:8081`
- **UDP Server**: `:8082` (listening), `:8083` (broadcast)
- **gRPC Server**: `:8084`
- **WebSocket**: `/ws` (on HTTP server)

## API Documentation

For detailed API documentation, see [API_DOCUMENTATION.md](API_DOCUMENTATION.md).

### Quick Start Examples

#### Register a User
```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"password123"}'
```

#### Login
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"password123"}'
```

#### Get All Manga
```bash
curl http://localhost:8080/api/v1/manga
```

#### Update Progress (requires authentication)
```bash
curl -X POST http://localhost:8080/api/v1/progress \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <your_token>" \
  -d '{"manga_id":"manga123","chapter":5}'
```

## Protocol Integration

The system demonstrates cross-protocol communication:

- **HTTP → TCP/UDP**: When progress is updated via HTTP API, it automatically broadcasts to all TCP and UDP clients
- **HTTP → UDP**: When a new manga is created via HTTP API, it broadcasts a notification to all UDP clients
- **Real-time Updates**: TCP and UDP clients receive real-time notifications of system events

## Performance

- Supports 50-100 concurrent users
- Handles 20-30 concurrent TCP connections
- WebSocket chat with 10-20 simultaneous users
- Search queries complete within 500ms
- Database supports 30-40 manga series

## Error Handling

All protocols implement comprehensive error handling:

- **Network Errors**: Timeouts, connection failures, and interruptions
- **Protocol Errors**: Invalid messages, malformed data
- **Client Cleanup**: Automatic removal of failed clients
- **Logging**: Detailed logging for debugging

## Security

- JWT authentication for protected endpoints
- Password hashing with bcrypt (cost factor 12)
- CORS configuration for web clients
- Input validation on all endpoints
- SQL injection prevention via parameterized queries

## Development

### Project Structure

- **cmd/**: Application entry points
- **internal/**: Private application code
- **pkg/**: Public library code
- **api/**: Protocol Buffer definitions
- **data/**: Initial data files

### Testing

The system is designed for demonstration and testing. All protocols can be tested independently:

- **HTTP**: Use curl, Postman, or any HTTP client
- **TCP**: Use `telnet` or `nc` (netcat) to connect to port 8081
- **UDP**: Use `nc -u` to connect to port 8082
- **WebSocket**: Use browser console or WebSocket client tools
- **gRPC**: Use gRPC client tools or generate client code from proto files

## License

This project is developed for educational purposes as part of a network-centric computing course.

## Authors

Developed for Network-Centric Computing Course, Semester 1, 2025-2026.
