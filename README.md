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

    Go >= 1.21

    protoc (Protocol Buffers compiler)

    protoc-gen-go and protoc-gen-go-grpc installed

### Run Services

  #### Stats Service

```bash
cd stats-service
go run main.go
```

  #### Fibonacci Service

```bash
cd fibonacci-service
go run main.go
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


## Potential Improvements

  - Server streaming Fibonacci sequences

  - Dockerize all services for easy deployment

  -  Persistent stats storage (Redis or database)

  -  Advanced metrics (cache hit/miss, max/min duration)

  -  HTTP API with query parameters for streaming, batch requests

## License

MIT License
