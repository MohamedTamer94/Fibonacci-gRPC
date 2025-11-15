package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	pb "fibonacci-grpc/proto/fibonacci"
	statsPb "fibonacci-grpc/proto/stats"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// client is the gRPC client for the Fibonacci service.
var client pb.FibonacciClient

// statsClient is the gRPC client for the Stats service.
var statsClient statsPb.StatsClient

// FibHandler handles HTTP requests to calculate the Fibonacci number for a given 'n'.
// Example request: GET /fib?n=10
func FibHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)

	nStr := r.URL.Query().Get("n")
	n, err := strconv.Atoi(nStr)
	if err != nil {
		log.Printf("Invalid input: %v", nStr)
		encoder.Encode(map[string]string{"error": "invalid integer"})
		return
	}
	if n < 0 {
		log.Printf("Negative input: %d", n)
		encoder.Encode(map[string]string{"error": "n must be non-negative"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, fibErr := client.GetFib(ctx, &pb.FibonacciRequest{N: int32(n)})
	if fibErr != nil {
		log.Printf("gRPC Fibonacci error: %v", fibErr)
		encoder.Encode(map[string]string{"error": fibErr.Error()})
		return
	}

	log.Printf("Fibonacci calculation for n=%d succeeded: %d", n, resp.GetX())
	encoder.Encode(resp)
}

// StatsHandler handles HTTP requests to retrieve service statistics.
// Example request: GET /stats
func StatsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, statsErr := statsClient.GetStats(ctx, nil)
	if statsErr != nil {
		log.Printf("gRPC Stats error: %v", statsErr)
		encoder.Encode(map[string]string{"error": statsErr.Error()})
		return
	}

	log.Println("Stats retrieval succeeded")
	encoder.Encode(resp)
}

// main initializes the gRPC clients and starts the HTTP API gateway server.
func main() {
	// Connect to Fibonacci gRPC service
	conn, err := grpc.Dial(":5001", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Fibonacci service: %v", err)
	}
	defer conn.Close()
	client = pb.NewFibonacciClient(conn)
	log.Println("Connected to Fibonacci gRPC service on :5001")

	// Connect to Stats gRPC service
	statsConn, statsErr := grpc.NewClient(":5002", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if statsErr != nil {
		log.Fatalf("Failed to connect to Stats service: %v", statsErr)
	}
	defer statsConn.Close()
	statsClient = statsPb.NewStatsClient(statsConn)
	log.Println("Connected to Stats gRPC service on :5002")

	// Register HTTP handlers
	http.HandleFunc("/fib", FibHandler)
	http.HandleFunc("/stats", StatsHandler)

	log.Println("API Gateway running on :3002")
	if httpErr := http.ListenAndServe(":3002", nil); httpErr != nil {
		log.Fatalf("Failed to start HTTP server: %v", httpErr)
	}
}
