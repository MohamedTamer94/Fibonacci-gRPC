package main

import (
	"context"
	"log"
	"net"
	"sync"
	"time"

	pb "fibonacci-grpc/proto/fibonacci"
	statsPb "fibonacci-grpc/proto/stats"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// fibonacciServer implements the Fibonacci gRPC service.
type fibonacciServer struct {
	pb.UnimplementedFibonacciServer
}

// statsClient is the gRPC client for sending statistics to the Stats service.
var statsClient statsPb.StatsClient

// fibCache caches previously computed Fibonacci numbers for efficiency.
var (
	fibCache = map[int]int{0: 0, 1: 1}
	mu       sync.RWMutex
)

// GetFib calculates the Fibonacci number for a given 'n'.
// It returns an error if 'n' is greater than 92 to prevent int64 overflow.
func (*fibonacciServer) GetFib(_ context.Context, r *pb.FibonacciRequest) (*pb.FibonacciResponse, error) {
	n := int(r.GetN())
	if n > 92 {
		log.Printf("Received too large n: %d", n)
		return nil, status.Error(codes.InvalidArgument, "n too large (max 92)")
	}

	start := time.Now()
	res := int64(Fib(n))
	duration := time.Since(start)

	log.Printf("Computed Fib(%d) = %d in %v", n, res, duration)

	// Fire-and-forget stats update
	go func(n int, dur time.Duration) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_, err := statsClient.RecordNo(ctx, &statsPb.RecordRequest{
			N:        int32(n),
			Duration: dur.Nanoseconds(),
		})
		if err != nil {
			log.Printf("Failed to record stats for n=%d: %v", n, err)
		} else {
			log.Printf("Recorded stats for n=%d, duration=%v", n, dur)
		}
	}(n, duration)

	return &pb.FibonacciResponse{X: res}, nil
}

// FibSlow calculates Fibonacci recursively without caching (for testing duration).
func FibSlow(n int) int {
	if n == 0 {
		return 0
	}
	if n == 1 {
		return 1
	}
	return FibSlow(n-1) + FibSlow(n-2)
}

// Fib calculates Fibonacci using a cache for performance.
func Fib(n int) int {
	if n == 0 {
		return 0
	}
	if n == 1 {
		return 1
	}

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

// main starts the Fibonacci gRPC server and connects to the Stats service.
func main() {
	// Connect to Stats gRPC service
	conn, statsErr := grpc.NewClient(":5002", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if statsErr != nil {
		log.Fatalf("Failed to connect to Stats service: %v", statsErr)
	}
	defer conn.Close()
	statsClient = statsPb.NewStatsClient(conn)
	log.Println("Connected to Stats gRPC service on :5002")

	// Start Fibonacci gRPC server
	lis, err := net.Listen("tcp", ":5001")
	if err != nil {
		log.Fatalf("Failed to listen on :5001: %v", err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterFibonacciServer(grpcServer, &fibonacciServer{})
	log.Println("Fibonacci gRPC server running on :5001")

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve gRPC server: %v", err)
	}
}
