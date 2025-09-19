package crawler

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// WorkerPool manages a pool of workers that process URLs concurrently
type WorkerPool struct {
	workers      []*Worker
	urlQueue     *URLQueue
	rateLimiter  *RateLimiter
	httpClient   *HTTPClient
	results      []CrawlResult
	resultsMutex sync.Mutex
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	
	// Performance metrics
	processedCount int64
	errorCount     int64
	startTime      time.Time
}

// Worker represents a single worker in the pool
type Worker struct {
	id           int
	pool         *WorkerPool
	processed    int64
	errors       int64
}

// NewWorkerPool creates a new worker pool with the specified number of workers
func NewWorkerPool(numWorkers int, urlQueue *URLQueue, rateLimiter *RateLimiter, httpClient *HTTPClient) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	
	pool := &WorkerPool{
		workers:     make([]*Worker, numWorkers),
		urlQueue:    urlQueue,
		rateLimiter: rateLimiter,
		httpClient:  httpClient,
		results:     make([]CrawlResult, 0),
		ctx:         ctx,
		cancel:      cancel,
		startTime:   time.Now(),
	}

	// Create workers
	for i := 0; i < numWorkers; i++ {
		pool.workers[i] = &Worker{
			id:   i,
			pool: pool,
		}
	}

	return pool
}

// Start begins all workers in the pool
func (p *WorkerPool) Start() {
	for _, worker := range p.workers {
		p.wg.Add(1)
		go worker.start()
	}
}

// Stop gracefully stops all workers
func (p *WorkerPool) Stop() {
	p.cancel()
	p.wg.Wait()
}

// Wait waits for all workers to complete
func (p *WorkerPool) Wait() {
	p.wg.Wait()
}

// GetResults returns a copy of all crawl results
func (p *WorkerPool) GetResults() []CrawlResult {
	p.resultsMutex.Lock()
	defer p.resultsMutex.Unlock()
	
	// Return a copy to prevent race conditions
	results := make([]CrawlResult, len(p.results))
	copy(results, p.results)
	return results
}

// GetMetrics returns performance metrics
func (p *WorkerPool) GetMetrics() WorkerMetrics {
	processed := atomic.LoadInt64(&p.processedCount)
	errors := atomic.LoadInt64(&p.errorCount)
	elapsed := time.Since(p.startTime)
	
	var urlsPerSecond float64
	if elapsed.Seconds() > 0 {
		urlsPerSecond = float64(processed) / elapsed.Seconds()
	}

	return WorkerMetrics{
		Processed:     processed,
		Errors:        errors,
		Elapsed:       elapsed,
		URLsPerSecond: urlsPerSecond,
		QueueSize:     p.urlQueue.Size(),
		VisitedCount:  p.urlQueue.VisitedCount(),
	}
}

// WorkerMetrics contains performance metrics for the worker pool
type WorkerMetrics struct {
	Processed     int64         `json:"processed"`
	Errors        int64         `json:"errors"`
	Elapsed       time.Duration `json:"elapsed"`
	URLsPerSecond float64       `json:"urls_per_second"`
	QueueSize     int           `json:"queue_size"`
	VisitedCount  int           `json:"visited_count"`
}

// start begins the worker's main processing loop
func (w *Worker) start() {
	defer w.pool.wg.Done()
	
	for {
		select {
		case <-w.pool.ctx.Done():
			return
		default:
			// Try to get a URL from the queue
			url, ok := w.pool.urlQueue.Dequeue()
			if !ok {
				// Queue is closed, worker should exit
				return
			}

			// Process the URL
			result := w.processURL(url)
			
			// Add result to the pool's results
			w.pool.addResult(result)
			
			// Update metrics
			atomic.AddInt64(&w.pool.processedCount, 1)
			if result.Error != "" {
				atomic.AddInt64(&w.pool.errorCount, 1)
				w.errors++
			} else {
				w.processed++
			}
		}
	}
}

// processURL processes a single URL with rate limiting and error handling
func (w *Worker) processURL(url URL) CrawlResult {
	startTime := time.Now()
	result := CrawlResult{
		URL:         url.URL,
		Depth:       url.Depth,
		ProcessedAt: startTime,
	}

	// Check if we've exceeded the maximum depth
	if url.Depth > 3 {
		result.Error = "maximum depth exceeded"
		result.ResponseTime = time.Since(startTime)
		return result
	}

	// Extract domain for rate limiting
	domain, err := ExtractDomain(url.URL)
	if err != nil {
		result.Error = fmt.Sprintf("invalid URL: %v", err)
		result.ResponseTime = time.Since(startTime)
		return result
	}

	// Apply rate limiting
	if err := w.pool.rateLimiter.Wait(w.pool.ctx, domain); err != nil {
		result.Error = fmt.Sprintf("rate limit error: %v", err)
		result.ResponseTime = time.Since(startTime)
		return result
	}

	// Fetch the URL
	resp, err := w.pool.httpClient.FetchURL(url.URL)
	if err != nil {
		result.Error = fmt.Sprintf("fetch error: %v", err)
		result.ResponseTime = time.Since(startTime)
		return result
	}

	result.Status = resp.StatusCode
	result.ResponseTime = time.Since(startTime)

	// Parse the HTML content
	title, links, err := ParseHTML(resp)
	if err != nil {
		result.Error = fmt.Sprintf("parse error: %v", err)
		return result
	}

	result.Title = title
	result.Links = links

	// Add new links to the queue if we haven't reached max depth
	if url.Depth < 3 {
		for _, link := range links {
			newURL := NewURL(link, url.Depth+1)
			// Ignore enqueue failures (queue might be closed or full)
			_ = w.pool.urlQueue.Enqueue(newURL)
		}
	}

	return result
}

// addResult adds a result to the pool's results slice (thread-safe)
func (p *WorkerPool) addResult(result CrawlResult) {
	p.resultsMutex.Lock()
	defer p.resultsMutex.Unlock()
	p.results = append(p.results, result)
}

// GetWorkerStats returns statistics for each worker
func (p *WorkerPool) GetWorkerStats() []WorkerStats {
	stats := make([]WorkerStats, len(p.workers))
	for i, worker := range p.workers {
		stats[i] = WorkerStats{
			ID:        worker.id,
			Processed: worker.processed,
			Errors:    worker.errors,
		}
	}
	return stats
}

// WorkerStats contains statistics for a single worker
type WorkerStats struct {
	ID        int   `json:"id"`
	Processed int64 `json:"processed"`
	Errors    int64 `json:"errors"`
}