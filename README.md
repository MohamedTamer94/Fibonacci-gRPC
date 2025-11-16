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
- **Redis Caching** with TTL ( for testing )
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
3. Fibonacci Service computes the number (or retrieves from redis cache).
4. Fibonacci Service sends **asynchronous stats update** to Stats Service.
5. Stats Service records total requests, per-number request counts, and average computation times.
6. Client receives JSON response.

---

## Features

- **Redis caching** for fast Fibonacci computation
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

    Go >= 1.21

    protoc (Protocol Buffers compiler)

    protoc-gen-go and protoc-gen-go-grpc installed

    Redis

### Run Services

  #### Stats Service

```bash
cd stats-service
go run main.go
```

  #### Fibonacci Service (Run multiple instances to test load balancing )

```bash
cd fibonacci-service
go run main.go --port=5001
go run main.go --port=5003
go run main.go --port=5005
```

  #### API Gateway

```bash
cd api-gateway
go run main.go
```

### Test the API

Get Fibonacci number:

```bash
curl "http://localhost:3002/fib?n=10"
```

Get Stats:

```bash
curl "http://localhost:3002/stats"
```

## Code Highlights

- Redis cache:
  ```go
       cached, err := rdb.Get(ctx, cacheKey).Result()
       if err == nil {
              // Cache hit
              log.Printf("Cache hit for Fib(%d) = %s", n, cached)
              cachedI, convErr := strconv.ParseInt(cached, 10, 64)
              if convErr != nil {
                     log.Printf("Failed to parse cached value: %v", convErr)
              } else {
                     return int(cachedI)
              }
       } else if err == redis.Nil {
              log.Printf("Cache miss for Fib(%d)", n)
       } else {
              log.Printf("Redis GET error: %v", err)
       }
       
       // Cache miss → compute
       a, b := 0, 1
       for i := 2; i <= n; i++ {
              a, b = b, a+b
       }
       // Store in Redis
       if err := rdb.Set(ctx, cacheKey, b, 0).Err(); err != nil {
              log.Printf("Failed to set cache: %v", err)
       }
       return b
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

  - Dockerize all services for easy deployment

  -  Persistent stats storage (Redis or database)

  -  Advanced metrics (cache hit/miss, max/min duration)

  -  HTTP API with query parameters for streaming, batch requests

## License

MIT License
