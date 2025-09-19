package crawler

import (
	"context"
	"testing"
	"time"
)

// BenchmarkURLQueue tests the performance of URL queue operations
func BenchmarkURLQueue(b *testing.B) {
	queue := NewURLQueue(1000)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			url := NewURL("https://example.com/test", 0)
			queue.Enqueue(url)
		}
	})
}

// BenchmarkRateLimiter tests the performance of rate limiting operations
func BenchmarkRateLimiter(b *testing.B) {
	rateLimiter := NewRateLimiter(10)
	defer rateLimiter.Stop()
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rateLimiter.Wait(ctx, "example.com")
	}
}

// BenchmarkHTTPClient tests the performance of HTTP client operations
func BenchmarkHTTPClient(b *testing.B) {
	client := NewHTTPClient(5*time.Second, 1)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Note: This will make actual HTTP requests, so it's more of an integration test
		// In a real benchmark, you'd want to mock the HTTP responses
		_, _ = client.FetchURL("https://httpbin.org/get")
	}
}

// BenchmarkWorkerPool tests the performance of worker pool operations
func BenchmarkWorkerPool(b *testing.B) {
	queue := NewURLQueue(1000)
	rateLimiter := NewRateLimiter(100) // High rate limit for benchmarking
	defer rateLimiter.Stop()
	client := NewHTTPClient(5*time.Second, 1)
	
	pool := NewWorkerPool(10, queue, rateLimiter, client)
	
	// Pre-populate queue with test URLs
	for i := 0; i < b.N; i++ {
		url := NewURL("https://httpbin.org/get", 0)
		queue.Enqueue(url)
	}
	
	b.ResetTimer()
	pool.Start()
	pool.Wait()
	pool.Stop()
}

// TestConcurrencySafety tests that the crawler is safe for concurrent access
func TestConcurrencySafety(t *testing.T) {
	queue := NewURLQueue(100)
	rateLimiter := NewRateLimiter(10)
	defer rateLimiter.Stop()
	client := NewHTTPClient(5*time.Second, 1)
	
	pool := NewWorkerPool(10, queue, rateLimiter, client)
	
	// Add URLs concurrently
	for i := 0; i < 100; i++ {
		go func(i int) {
			url := NewURL("https://httpbin.org/get", 0)
			queue.Enqueue(url)
		}(i)
	}
	
	// Start workers
	pool.Start()
	
	// Let it run for a short time
	time.Sleep(100 * time.Millisecond)
	
	// Stop gracefully
	queue.Close()
	pool.Stop()
	
	// Check that no race conditions occurred
	results := pool.GetResults()
	if len(results) == 0 {
		t.Error("Expected some results from concurrent processing")
	}
}
