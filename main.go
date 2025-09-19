package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-concurrent-web-crawler/crawler"
)

// Configuration constants
const (
	DefaultStartURL      = "https://httpbin.org"
	DefaultMaxDepth      = 3
	DefaultNumWorkers    = 10
	DefaultQueueSize     = 100
	DefaultRateLimit     = 10 // requests per second per domain
	DefaultTimeout       = 30 * time.Second
	DefaultRetries       = 3
	DefaultProgressInterval = 2 * time.Second
)

// Config holds all configuration options for the crawler
type Config struct {
	StartURL         string
	MaxDepth         int
	NumWorkers       int
	QueueSize        int
	RateLimit        int
	Timeout          time.Duration
	Retries          int
	OutputFile       string
	ProgressInterval time.Duration
	MaxURLs          int
	Verbose          bool
}

// parseFlags parses command line flags and returns configuration
func parseFlags() *Config {
	config := &Config{}

	flag.StringVar(&config.StartURL, "url", DefaultStartURL, "Starting URL to crawl")
	flag.IntVar(&config.MaxDepth, "depth", DefaultMaxDepth, "Maximum crawl depth")
	flag.IntVar(&config.NumWorkers, "workers", DefaultNumWorkers, "Number of concurrent workers")
	flag.IntVar(&config.QueueSize, "queue", DefaultQueueSize, "URL queue buffer size")
	flag.IntVar(&config.RateLimit, "rate", DefaultRateLimit, "Requests per second per domain")
	flag.DurationVar(&config.Timeout, "timeout", DefaultTimeout, "HTTP request timeout")
	flag.IntVar(&config.Retries, "retries", DefaultRetries, "Number of retries for failed requests")
	flag.StringVar(&config.OutputFile, "output", "examples/output.json", "Output JSON file path")
	flag.DurationVar(&config.ProgressInterval, "progress", DefaultProgressInterval, "Progress reporting interval")
	flag.IntVar(&config.MaxURLs, "max", 0, "Maximum number of URLs to process (0 = unlimited)")
	flag.BoolVar(&config.Verbose, "verbose", false, "Enable verbose logging")

	flag.Parse()

	return config
}

// validateConfig validates the configuration and sets defaults
func validateConfig(config *Config) error {
	if config.StartURL == "" {
		return fmt.Errorf("start URL cannot be empty")
	}
	if config.MaxDepth < 0 || config.MaxDepth > 10 {
		return fmt.Errorf("max depth must be between 0 and 10")
	}
	if config.NumWorkers < 1 || config.NumWorkers > 100 {
		return fmt.Errorf("number of workers must be between 1 and 100")
	}
	if config.QueueSize < 1 {
		return fmt.Errorf("queue size must be at least 1")
	}
	if config.RateLimit < 1 {
		return fmt.Errorf("rate limit must be at least 1")
	}
	if config.Timeout < time.Second {
		return fmt.Errorf("timeout must be at least 1 second")
	}
	if config.Retries < 0 {
		return fmt.Errorf("retries cannot be negative")
	}

	return nil
}

func main() {
	// Parse command line flags
	config := parseFlags()

	// Validate configuration
	if err := validateConfig(config); err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Print startup information
	fmt.Printf("🕷️  Go Concurrent Web Crawler\n")
	fmt.Printf("═══════════════════════════════════════\n")
	fmt.Printf("Starting URL: %s\n", config.StartURL)
	fmt.Printf("Max Depth: %d\n", config.MaxDepth)
	fmt.Printf("Workers: %d\n", config.NumWorkers)
	fmt.Printf("Rate Limit: %d req/sec/domain\n", config.RateLimit)
	fmt.Printf("Queue Size: %d\n", config.QueueSize)
	fmt.Printf("Timeout: %v\n", config.Timeout)
	fmt.Printf("Max URLs: %d\n", config.MaxURLs)
	fmt.Printf("Output: %s\n", config.OutputFile)
	fmt.Printf("═══════════════════════════════════════\n\n")

	// Create context with cancellation for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Printf("\n🛑 Received shutdown signal, stopping gracefully...\n")
		cancel()
	}()

	// Initialize crawler components
	urlQueue := crawler.NewURLQueue(config.QueueSize)
	rateLimiter := crawler.NewRateLimiter(config.RateLimit)
	httpClient := crawler.NewHTTPClient(config.Timeout, config.Retries)

	// Create worker pool
	workerPool := crawler.NewWorkerPool(config.NumWorkers, urlQueue, rateLimiter, httpClient)

	// Create progress reporter
	progressReporter := crawler.NewProgressReporter(workerPool, config.ProgressInterval)

	// Start progress reporting
	progressReporter.Start()
	defer progressReporter.Stop()

	// Start worker pool
	workerPool.Start()
	defer workerPool.Stop()

	// Enqueue the starting URL
	startURL := crawler.NewURL(config.StartURL, 0)
	if !urlQueue.Enqueue(startURL) {
		log.Fatalf("Failed to enqueue starting URL: %s", config.StartURL)
	}

	// Start a goroutine to monitor for completion
	done := make(chan struct{})
	go func() {
		defer close(done)
		
		// Wait for either context cancellation or queue to be empty
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Check if queue is empty and no workers are active
				if urlQueue.IsEmpty() && urlQueue.Size() == 0 {
					// Give workers a moment to finish processing
					time.Sleep(2 * time.Second)
					if urlQueue.IsEmpty() && urlQueue.Size() == 0 {
						return
					}
				}
				
				// Check if we've reached the maximum number of URLs
				if config.MaxURLs > 0 && urlQueue.VisitedCount() >= config.MaxURLs {
					fmt.Printf("\n🎯 Reached maximum URL limit (%d), stopping...\n", config.MaxURLs)
					return
				}
			}
		}
	}()

	// Wait for completion or cancellation
	select {
	case <-ctx.Done():
		fmt.Printf("\n⏹️  Crawling cancelled by user\n")
	case <-done:
		fmt.Printf("\n✅ Crawling completed naturally\n")
	}

	// Close the URL queue to signal workers to stop
	urlQueue.Close()

	// Wait for all workers to finish
	workerPool.Wait()

	// Get final results
	results := workerPool.GetResults()
	metrics := workerPool.GetMetrics()

	// Save results to JSON file
	if err := saveResults(results, config.OutputFile); err != nil {
		log.Printf("Error saving results: %v", err)
	}

	// Print final summary
	printFinalSummary(results, metrics, config)

	// Clean up rate limiter
	rateLimiter.Stop()
}

// saveResults saves the crawl results to a JSON file
func saveResults(results []crawler.CrawlResult, filename string) error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll("examples", 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Create output file
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer file.Close()

	// Create JSON encoder with indentation
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	// Encode results
	if err := encoder.Encode(results); err != nil {
		return fmt.Errorf("failed to encode results: %v", err)
	}

	return nil
}

// printFinalSummary prints a comprehensive summary of the crawling session
func printFinalSummary(results []crawler.CrawlResult, metrics crawler.WorkerMetrics, config *Config) {
	fmt.Printf("\n📊 Final Summary\n")
	fmt.Printf("═══════════════════════════════════════\n")
	
	// Basic statistics
	fmt.Printf("📈 Performance Metrics:\n")
	fmt.Printf("  • Total URLs processed: %d\n", metrics.Processed)
	fmt.Printf("  • Total errors: %d\n", metrics.Errors)
	fmt.Printf("  • Success rate: %.1f%%\n", 
		float64(metrics.Processed-metrics.Errors)/float64(metrics.Processed)*100)
	fmt.Printf("  • Average URLs/second: %.2f\n", metrics.URLsPerSecond)
	fmt.Printf("  • Total time: %v\n", metrics.Elapsed)
	
	// Results breakdown
	successfulResults := 0
	totalLinks := 0
	depthCounts := make(map[int]int)
	
	for _, result := range results {
		if result.Error == "" {
			successfulResults++
			totalLinks += len(result.Links)
		}
		depthCounts[result.Depth]++
	}
	
	fmt.Printf("\n📋 Results Breakdown:\n")
	fmt.Printf("  • Successful crawls: %d\n", successfulResults)
	fmt.Printf("  • Total links found: %d\n", totalLinks)
	fmt.Printf("  • Average links per page: %.1f\n", 
		float64(totalLinks)/float64(successfulResults))
	
	fmt.Printf("\n📊 Depth Distribution:\n")
	for depth := 0; depth <= config.MaxDepth; depth++ {
		count := depthCounts[depth]
		fmt.Printf("  • Depth %d: %d URLs\n", depth, count)
	}
	
	// Configuration used
	fmt.Printf("\n⚙️  Configuration Used:\n")
	fmt.Printf("  • Workers: %d\n", config.NumWorkers)
	fmt.Printf("  • Rate limit: %d req/sec/domain\n", config.RateLimit)
	fmt.Printf("  • Max depth: %d\n", config.MaxDepth)
	fmt.Printf("  • Timeout: %v\n", config.Timeout)
	fmt.Printf("  • Retries: %d\n", config.Retries)
	
	fmt.Printf("\n💾 Output saved to: %s\n", config.OutputFile)
	fmt.Printf("═══════════════════════════════════════\n")
}