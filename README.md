# Fibonacci gRPC Microservices Project

![Go](https://img.shields.io/badge/Go-1.21-blue)
![gRPC](https://img.shields.io/badge/gRPC-enabled-brightgreen)

## Overview

This project demonstrates a **mini distributed system** implemented in Go using **gRPC**, **concurrent caching**, and **service-to-service communication**.  
It consists of three main components:

1. **Fibonacci Service** – computes Fibonacci numbers efficiently with an in-memory cache.
2. **Stats Service** – collects metrics about Fibonacci requests, including request counts and average computation times.
3. **API Gateway** – exposes HTTP endpoints for clients and communicates with both gRPC services.

The system showcases:

- **gRPC unary RPCs** for service communication
- **In-memory caching** with thread-safe access
- **Fire-and-forget asynchronous stats updates**
- **Concurrency and mutex handling**
- **Retry logic for transient network failures**
- **Structured logging**

---

## Architecture

       +----------------+
       |   API Gateway  |
       | /fib  /stats   |
       +--------+-------+
                |
     gRPC client|
                v
       +----------------+
       | Fibonacci      |
       | Service        |
       +--------+-------+
                |
    Fire-and-forget stats RPC
                v
       +----------------+
       | Stats Service  |
       +----------------+


**Request flow:**

1. Client calls `/fib?n=10` → API Gateway.
2. API Gateway forwards gRPC request to Fibonacci Service.
3. Fibonacci Service computes the number (or retrieves from cache).
4. Fibonacci Service sends **asynchronous stats update** to Stats Service.
5. Stats Service records total requests, per-number request counts, and average computation times.
6. Client receives JSON response.

---

## Features

- **Thread-safe in-memory caching** for fast Fibonacci computation
- **Stats collection**: total requests, per-number request count, average computation time
- **Fire-and-forget stats updates** to minimize response latency
- **Retries with exponential backoff** for transient network errors
- **HTTP API Gateway** exposing `/fib` and `/stats` endpoints
- **gRPC proto definitions** for clean, type-safe communication
- **Structured logging** for requests, cache hits, and stats updates

---

## Proto Definitions

### Fibonacci Service (`proto/fibonacci/fib.proto`)

```proto
service Fibonacci {
    rpc GetFib(FibonacciRequest) returns (FibonacciResponse);
}

message FibonacciRequest {
    int32 n = 1;
}

message FibonacciResponse {
    int64 x = 1;
}
```

### Stats Service (proto/stats/stats.proto)

```proto
service Stats {
    rpc RecordNo(RecordRequest) returns (RecordResponse);
    rpc GetStats(google.protobuf.Empty) returns (StatsResponse);
}

message StatsResponse {
    int32 total_requests = 1;
    repeated FibonacciStat fibonacci_stats = 2;
}

message FibonacciStat {
    int32 n = 1;
    int32 request_count = 2;
    double average_time_ms = 3;
}

message RecordRequest {
    int32 n = 1;
    int64 duration = 2;
}

message RecordResponse {
    bool success = 1;
}
```

## Getting Started
### Requirements

- Go >= 1.21 (for local development)
- protoc (Protocol Buffers compiler) if you regenerate protos locally
- protoc-gen-go and protoc-gen-go-grpc (optional for local proto generation)
- Docker & Docker Compose (recommended for running the whole system in containers)

### Run Services

You can run the system either locally (development mode) or using Docker Compose (recommended for integration / deployment).

Option A — run locally (development)

1. Start each service in its folder using `go run`:

```powershell
cd stats-service; go run main.go
cd fibonacci-service; go run main.go
cd api-gateway; go run main.go
```

Option B — run with Docker Compose (recommended)

The repository includes Dockerfiles for each service and a `docker-compose.yaml` at the project root that brings up all services plus a Redis container used by the Fibonacci service.

From the project root (where `docker-compose.yaml` lives), build and start everything with:

```powershell
# build images and start services in the foreground
docker compose up --build

# or run in detached/background mode
docker compose up --build -d
```

Notes:
- The compose file exposes the API Gateway on port 3002 (host) mapped to 3002 (container).
- Redis is provided as a separate container and mounted to a named volume (`redis-data`).
- The services use the following ports inside the Docker network:
  - fibonacci-service: 5001
  - stats-service: 5002

To stop and tear down (removes containers, networks; keeps named volumes by default):

```powershell
docker compose down
```

To remove named volumes as well (e.g., to clear Redis data):

```powershell
docker compose down -v
```

## Code Highlights

- Thread-safe cache:
  ```go
  var fibCache = map[int]int{0:0, 1:1}
  var mu sync.RWMutex

  func Fib(n int) int {
    mu.RLock()
    cached, ok := fibCache[n]
    mu.RUnlock()
    if ok {
        return cached
    }
    mu.Lock()
    for i := 2; i <= n; i++ {
        if _, exists := fibCache[i]; !exists {
            fibCache[i] = fibCache[i-1] + fibCache[i-2]
        }
    }
    result := fibCache[n]
    mu.Unlock()
    return result
  }
  ```
- Fire-and-forget stats update with retries:
  ```go
  go func(n int, dur time.Duration) {
    err := RetryGRPC(3, 100*time.Millisecond, func() error {
        ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
        defer cancel()
        _, err := statsClient.RecordNo(ctx, &statsPb.RecordRequest{
            N: int32(n),
            Duration: dur.Nanoseconds(),
        })
        return err
    })
    if err != nil {
        log.Printf("Failed to record stats for n=%d: %v", n, err)
    }
  }(n, duration)
  ```

## Potential Improvements

- Server streaming Fibonacci sequences

- Persistent stats storage (long-term storage for metrics: Redis/DB)

- Advanced metrics (cache hit/miss, max/min duration, percentiles)

- Add mTLS / authentication between services

Note: Docker Compose + Dockerfiles are included in the repository now so the whole stack can be run locally or in CI via the same containerized setup.

## License

MIT License
