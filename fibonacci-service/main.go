package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"

	pb "fibonacci-grpc/proto/fibonacci"
	statsPb "fibonacci-grpc/proto/stats"

	"github.com/redis/go-redis/v9"
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

// command line flag for determining the port in which this app will run on ( used for load balancing tests )
var (
	portPtr = flag.Int("port", 5001, "specify the port that the app will run on")
)

// the client for redis; used for caching
var rdb *redis.Client

// background context for redis
var ctx = context.Background()

// initialize the redis client
func InitRedis() {
	rdb = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Redis not reachable: %v", err)
	}

}

func RetryGRPC(maxRetries int, baseDelay time.Duration, f func() error) error {
	var err error
	delay := baseDelay

	for i := 0; i <= maxRetries; i++ {
		err = f()
		if err == nil {
			return nil
		}

		// Check if error is retryable
		st, ok := status.FromError(err)
		if !ok {
			return err // non-gRPC error
		}
		if st.Code() != codes.Unavailable && st.Code() != codes.DeadlineExceeded {
			return err // permanent error, don't retry
		}

		// Retry after delay
		time.Sleep(delay)
		delay *= 2 // exponential backoff
	}

	return err // return last error if all retries fail
}

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
		err := RetryGRPC(3, 100*time.Millisecond, func() error {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_, err := statsClient.RecordNo(ctx, &statsPb.RecordRequest{
				N:        int32(n),
				Duration: dur.Nanoseconds(),
			})
			return err
		})
		if err != nil {
			// optional: log the error
			log.Printf("Failed to record stats for n=%d: %v", n, err)
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

	cacheKey := fmt.Sprintf("fib:%d", n)
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

}

// main starts the Fibonacci gRPC server and connects to the Stats service.
func main() {
	// parse command line flags
	flag.Parse()
	// initialize Redis DB for caching
	InitRedis()
	// Connect to Stats gRPC service
	conn, statsErr := grpc.NewClient(":5002", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if statsErr != nil {
		log.Fatalf("Failed to connect to Stats service: %v", statsErr)
	}
	defer conn.Close()
	statsClient = statsPb.NewStatsClient(conn)
	log.Println("Connected to Stats gRPC service on :5002")

	// Start Fibonacci gRPC server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *portPtr))
	if err != nil {
		log.Fatalf("Failed to listen on :5001: %v", err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterFibonacciServer(grpcServer, &fibonacciServer{})
	log.Printf("Fibonacci gRPC server running on :%d\n", *portPtr)

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve gRPC server: %v", err)
	}
}
