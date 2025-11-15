package main

import (
	"context"
	"log"
	"net"
	"sort"
	"sync"
	"time"

	pb "fibonacci-grpc/proto/stats"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

// statsService implements the Stats gRPC service.
type statsService struct {
	pb.UnimplementedStatsServer
	stats *Stats
}

// Stats stores aggregated statistics for Fibonacci requests.
type Stats struct {
	mu            sync.Mutex
	RequestCount  map[int]int           // Number of requests per 'n'
	TotalRequests int                   // Total number of requests
	TotalTime     map[int]time.Duration // Total processing time per 'n'
}

// RecordNo records a Fibonacci request and its duration.
// This method is called by the Fibonacci service asynchronously.
func (s *statsService) RecordNo(_ context.Context, r *pb.RecordRequest) (*pb.RecordResponse, error) {
	s.stats.mu.Lock()
	defer s.stats.mu.Unlock()

	n := int(r.GetN())
	dur := time.Duration(r.GetDuration())

	s.stats.RequestCount[n]++
	s.stats.TotalRequests++
	s.stats.TotalTime[n] += dur

	log.Printf("Recorded request for n=%d, duration=%v", n, dur)
	return &pb.RecordResponse{Success: true}, nil
}

// GetStats returns aggregated Fibonacci statistics, including request counts and average times.
func (s *statsService) GetStats(_ context.Context, in *emptypb.Empty) (*pb.StatsResponse, error) {
	var res []*pb.FibonacciStat

	s.stats.mu.Lock()
	defer s.stats.mu.Unlock()

	// Collect keys and sort
	keys := make([]int, 0, len(s.stats.RequestCount))
	for k := range s.stats.RequestCount {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	// Build sorted stats response
	for _, n := range keys {
		count := s.stats.RequestCount[n]
		res = append(res, &pb.FibonacciStat{
			N:            int32(n),
			RequestCount: int32(count),
			AverageTimeMs: float64(s.stats.TotalTime[n].Milliseconds()) / float64(count),
		})
	}

	log.Printf("Returning stats: total requests=%d, tracked values=%d", s.stats.TotalRequests, len(keys))
	return &pb.StatsResponse{
		TotalRequests:  int32(s.stats.TotalRequests),
		FibonacciStats: res,
	}, nil
}

// main starts the Stats gRPC server on port 5002.
func main() {
	lis, err := net.Listen("tcp", ":5002")
	if err != nil {
		log.Fatalf("Failed to listen on :5002: %v", err)
	}

	server := grpc.NewServer()

	defaultStats := &Stats{
		RequestCount: make(map[int]int),
		TotalTime:    make(map[int]time.Duration),
	}

	pb.RegisterStatsServer(server, &statsService{stats: defaultStats})

	log.Println("Stats gRPC server running on :5002")
	if err := server.Serve(lis); err != nil {
		log.Fatalf("Failed to serve gRPC server: %v", err)
	}
}
