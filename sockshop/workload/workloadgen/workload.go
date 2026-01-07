package workloadgen

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"flag"

	"github.com/blueprint-uservices/blueprint/examples/sockshop/workflow/frontend"
)

// The WorkloadGen interface, which the Blueprint compiler will treat as a
// Workflow service
type SimpleWorkload interface {
	ImplementsSimpleWorkload(context.Context) error
}

// workloadGen implementation
type workloadGen struct {
	SimpleWorkload

	frontend frontend.Frontend
	
	// Performance metrics
	mu        sync.Mutex
	latencies []time.Duration
	requests  int64
	errors    int64
}

var myarg = flag.Int("myarg", 12345, "help message for myarg")
var duration = flag.Int("duration", 60, "duration of workload in seconds")
var workers = flag.Int("workers", 100, "number of concurrent workers (0 for rate-limited mode)")
var rate = flag.Int("rate", 0, "requests per second (only used if workers=0)")

func NewSimpleWorkload(ctx context.Context, frontend frontend.Frontend) (SimpleWorkload, error) {
	return &workloadGen{
		frontend:  frontend,
		latencies: make([]time.Duration, 0, 10000),
	}, nil
}

func (s *workloadGen) Run(ctx context.Context) error {
	_, err := s.frontend.LoadCatalogue(ctx)
	if err != nil {
		fmt.Println("Failed to load catalogue")
		return err
	}
	
	if *workers > 0 {
		fmt.Printf("Starting workload generator (Max Throughput Mode):\n")
		fmt.Printf("  Duration: %d seconds\n", *duration)
		fmt.Printf("  Workers: %d\n", *workers)
		fmt.Println()
		return s.runMaxThroughput(ctx)
	} else {
		fmt.Printf("Starting workload generator (Rate Limited Mode):\n")
		fmt.Printf("  Duration: %d seconds\n", *duration)
		fmt.Printf("  Rate: %d req/s\n", *rate)
		fmt.Println()
		return s.runRateLimited(ctx)
	}
}

func (s *workloadGen) runMaxThroughput(ctx context.Context) error {
	startTime := time.Now()
	endTime := startTime.Add(time.Duration(*duration) * time.Second)
	
	// Print stats every 5 seconds
	statsTicker := time.NewTicker(5 * time.Second)
	defer statsTicker.Stop()
	
	// Create a context that will be cancelled after duration
	workCtx, cancel := context.WithDeadline(ctx, endTime)
	defer cancel()
	
	// Start worker goroutines
	var wg sync.WaitGroup
	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-workCtx.Done():
					return
				default:
					s.executeRequest(workCtx)
				}
			}
		}()
	}
	
	// Print stats periodically
	go func() {
		for {
			select {
			case <-workCtx.Done():
				return
			case <-statsTicker.C:
				s.printStats(startTime)
			}
		}
	}()
	
	// Wait for all workers to finish
	wg.Wait()
	s.printFinalStats(startTime)
	return nil
}

func (s *workloadGen) runRateLimited(ctx context.Context) error {
	startTime := time.Now()
	endTime := startTime.Add(time.Duration(*duration) * time.Second)
	ticker := time.NewTicker(time.Second / time.Duration(*rate))
	defer ticker.Stop()
	
	// Print stats every 5 seconds
	statsTicker := time.NewTicker(5 * time.Second)
	defer statsTicker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			s.printFinalStats(startTime)
			return nil
		case now := <-ticker.C:
			if now.After(endTime) {
				s.printFinalStats(startTime)
				return nil
			}
			go s.executeRequest(ctx)
		case <-statsTicker.C:
			s.printStats(startTime)
		}
	}
}

func (s *workloadGen) executeRequest(ctx context.Context) {
	start := time.Now()
	_, err := s.frontend.ListItems(ctx, []string{}, "", 1, 100)
	latency := time.Since(start)
	
	s.mu.Lock()
	s.requests++
	if err != nil {
		s.errors++
	} else {
		s.latencies = append(s.latencies, latency)
	}
	s.mu.Unlock()
}

func (s *workloadGen) printStats(startTime time.Time) {
	s.mu.Lock()
	elapsed := time.Since(startTime)
	requests := s.requests
	errors := s.errors
	latenciesCopy := make([]time.Duration, len(s.latencies))
	copy(latenciesCopy, s.latencies)
	s.mu.Unlock()
	
	if len(latenciesCopy) == 0 {
		return
	}
	
	sort.Slice(latenciesCopy, func(i, j int) bool {
		return latenciesCopy[i] < latenciesCopy[j]
	})
	
	throughput := float64(requests) / elapsed.Seconds()
	avg := average(latenciesCopy)
	p50 := percentile(latenciesCopy, 50)
	p95 := percentile(latenciesCopy, 95)
	p99 := percentile(latenciesCopy, 99)
	
	fmt.Printf("[%.1fs] Requests: %d | Errors: %d | Throughput: %.1f req/s | Avg: %v | p50: %v | p95: %v | p99: %v\n",
		elapsed.Seconds(), requests, errors, throughput, avg, p50, p95, p99)
}

func (s *workloadGen) printFinalStats(startTime time.Time) {
	fmt.Println("\n=== Final Results ===")
	s.printStats(startTime)
}

func average(latencies []time.Duration) time.Duration {
	if len(latencies) == 0 {
		return 0
	}
	var sum time.Duration
	for _, l := range latencies {
		sum += l
	}
	return sum / time.Duration(len(latencies))
}

func percentile(sorted []time.Duration, p int) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	idx := (len(sorted) * p) / 100
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func (s *workloadGen) ImplementsSimpleWorkload(context.Context) error {
	return nil
}
