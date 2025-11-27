package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
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
	// environment variable to determine the port that the app will run on
	port := os.Getenv("PORT")
	if port == "" {
		port = "5002" // default
	}
	fibUrl := os.Getenv("FIBONACCI_SERVICE_URL")
	statsUrl := os.Getenv("STATS_SERVICE_URL")
	log.Printf("API Gateway starting on port :%s", port)
	log.Printf("Fibonacci Service URL: %s", fibUrl)
	log.Printf("Stats Service URL: %s", statsUrl)
	// Connect to Fibonacci gRPC service ( the load balancer made by nginx is available on 8081 as per as nginx.conf )
	conn, err := grpc.NewClient(fibUrl, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Fibonacci service: %v", err)
	}
	defer conn.Close()
	client = pb.NewFibonacciClient(conn)
	log.Printf("Connected to Fibonacci gRPC service on %s\n", fibUrl)

	// Connect to Stats gRPC service
	statsConn, statsErr := grpc.NewClient(statsUrl, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if statsErr != nil {
		log.Fatalf("Failed to connect to Stats service: %v", statsErr)
	}
	defer statsConn.Close()
	statsClient = statsPb.NewStatsClient(statsConn)
	log.Printf("Connected to Stats gRPC service on :%s\n", statsUrl)

	// Register HTTP handlers
	http.HandleFunc("/fib", FibHandler)
	http.HandleFunc("/stats", StatsHandler)

	log.Printf("API Gateway running on :%d\n", port)
	if httpErr := http.ListenAndServe(":"+port, nil); httpErr != nil {
		log.Fatalf("Failed to start HTTP server: %v", httpErr)
	}
}
