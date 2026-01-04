# Gedis - Go Redis Clone

**Gedis** is a lightweight, high-performance Redis clone written in Go. It allows you to run a key-value store that speaks the Redis Serialization Protocol (RESP), making it compatible with standard Redis clients and libraries.

This project was built for educational purposes to understand the internals of database systems, network programming, and the RESP protocol.

## Features

- **RESP Protocol Support**: Fully compatible with Redis clients (`redis-cli`, `go-redis`, etc.).
- **Concurrent Networking**: Handles multiple client connections simultaneously using Go routines.
- **In-Memory Storage**: Thread-safe key-value store protected by mutexes.
- **Persistence (AOF)**: Durability via Append-Only File (AOF). Data survives server restarts.
- **Key Expiry (TTL)**: Support for `EX`, `PX` options and active/passive key expiration.
- **Pub/Sub System**: Real-time messaging with `SUBSCRIBE` and `PUBLISH`.
- **Compatibility Layer**: Handlers for `COMMAND`, `INFO`, `CLIENT`, and `SELECT` to support standard libraries.

##  Project Status

### Implemented Functionality
- [x] **Networking**: TCP Server listening on configurable ports.
- [x] **Protocol**: Full RESP Parser (Strings, Errors, Integers, Bulk Strings, Arrays) and Writer.
- [x] **Core Commands**: `PING`, `ECHO`, `SET`, `GET`.
- [x] **Key Management**: `DEL`, `EXISTS`.
- [x] **Expiry**: `TTL` command, `SET` with expiration, background cleanup job.
- [x] **Persistence**: AOF logging and startup replay.
- [x] **Pub/Sub**: `SUBSCRIBE`, `PUBLISH`.
- [x] **Pipeline Support**: Inherently supported.

## Getting Started

### Prerequisites
- [Go](https://go.dev/dl/) 1.21 or higher.
- [redis-cli](https://redis.io/docs/install/install-redis/) (optional, for testing).

###  Running Locally

1.  **Clone the repository:**
    ```bash
    git clone https://github.com/lucabrx/gedis.git
    cd gedis
    ```

2.  **Run the server:**
    ```bash
    # Default port 6379, saves to database.aof
    go run .
    
    # Custom port and AOF file
    go run . -port 6380 -aof mydata.aof
    ```

### 🐳 Running with Docker

1.  **Build the image:**
    ```bash
    docker build -t gedis .
    ```

2.  **Run the container:**
    ```bash
    # Map host port 6379 to container port 6379
    docker run -p 6379:6379 -v $(pwd)/data:/data gedis
    ```
    *Note: The `-v` flag persists your data to a local `data` folder.*

## 🎮 Usage Guide

You can connect to **Gedis** using `redis-cli` or any Redis client library.

### Basic Commands
```bash
$ redis-cli -p 6379

127.0.0.1:6379> PING
"PONG"

127.0.0.1:6379> SET user:1 "Alice"
OK

127.0.0.1:6379> GET user:1
"Alice"

127.0.0.1:6379> EXISTS user:1
(integer) 1

127.0.0.1:6379> DEL user:1
(integer) 1
```

### Expiry (TTL)
```bash
# Set key to expire in 10 seconds
127.0.0.1:6379> SET session "valid" EX 10
OK

127.0.0.1:6379> TTL session
(integer) 8

# ... wait ...
127.0.0.1:6379> GET session
(nil)
```

### Pub/Sub
**Terminal 1 (Subscriber):**
```bash
127.0.0.1:6379> SUBSCRIBE news
Reading messages... (press Ctrl-C to quit)
1) "subscribe"
2) "news"
3) (integer) 1
```

**Terminal 2 (Publisher):**
```bash
127.0.0.1:6379> PUBLISH news "Breaking News!"
(integer) 1
```

## Configuration

The server accepts the following command-line flags:

| Flag | Default | Description |
| :--- | :--- | :--- |
| `-port` | `6379` | The TCP port to listen on. |
| `-aof` | `database.aof` | The path to the Append-Only File for persistence. |

## Architecture Notes

- **AOF Persistence**: Write operations (`SET`, `DEL`) are appended to the AOF file immediately. On startup, the server reads this file and replays the commands to restore the in-memory state.
- **Graceful Shutdown**: The server catches `SIGINT` (Ctrl+C) and `SIGTERM` to close the AOF file properly before exiting, ensuring data integrity.
- **Compatibility**: We implement dummy handlers for `COMMAND`, `INFO`, etc., so standard libraries (like `go-redis`, `redis-py`) can connect without handshaking errors.

## AOF (Append Only File)

**AOF**, or **Append Only File**, is a persistence technique used by Redis (and Gedis) to ensure data durability.

### How it works
Instead of taking snapshots of the entire database state (like RDB persistence), AOF works by logging every **write operation** (like `SET` or `DEL`) received by the server into a file.

1.  **Logging**: When you run `SET key value`, the server writes that command to the end of the `database.aof` file.
2.  **Replaying**: When the server restarts, it reads the AOF file from top to bottom and re-executes all the commands in order. This reconstructs the database exactly as it was before the shutdown.

### Why is it useful?
*   **Durability**: Since every change is logged immediately, you lose minimal data in case of a crash.
*   **Simplicity**: The file format is just the standard RESP protocol, making it easy to parse and debug (you can actually open the `.aof` file in a text editor to see your commands!).
