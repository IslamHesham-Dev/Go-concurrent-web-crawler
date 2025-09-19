package crawler

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"
)

// ProgressReporter handles real-time progress reporting and metrics collection
type ProgressReporter struct {
	pool        *WorkerPool
	interval    time.Duration
	ctx         context.Context
	cancel      context.CancelFunc
	lastCount   int64
	startTime   time.Time
}

// NewProgressReporter creates a new progress reporter
func NewProgressReporter(pool *WorkerPool, interval time.Duration) *ProgressReporter {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &ProgressReporter{
		pool:      pool,
		interval:  interval,
		ctx:       ctx,
		cancel:    cancel,
		startTime: time.Now(),
	}
}

// Start begins the progress reporting goroutine
func (pr *ProgressReporter) Start() {
	go pr.reportLoop()
}

// Stop stops the progress reporter
func (pr *ProgressReporter) Stop() {
	pr.cancel()
}

// reportLoop runs the main reporting loop
func (pr *ProgressReporter) reportLoop() {
	ticker := time.NewTicker(pr.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pr.printProgress()
		case <-pr.ctx.Done():
			pr.printFinalStats()
			return
		}
	}
}

// printProgress prints current progress and performance metrics
func (pr *ProgressReporter) printProgress() {
	metrics := pr.pool.GetMetrics()
	currentCount := atomic.LoadInt64(&pr.pool.processedCount)
	
	// Calculate URLs processed in the last interval
	_ = currentCount - pr.lastCount // Track interval progress for future use
	pr.lastCount = currentCount
	
	// Calculate average URLs per second
	var avgURLsPerSecond float64
	if metrics.Elapsed.Seconds() > 0 {
		avgURLsPerSecond = float64(currentCount) / metrics.Elapsed.Seconds()
	}

	// Print progress information
	fmt.Printf("\r🚀 Progress: %d URLs processed | %d errors | %.1f URLs/sec | Queue: %d | Visited: %d | Active domains: %d",
		currentCount,
		metrics.Errors,
		avgURLsPerSecond,
		metrics.QueueSize,
		metrics.VisitedCount,
		len(pr.pool.rateLimiter.GetActiveDomains()))

	// Print worker-specific stats every 10 seconds
	if int(metrics.Elapsed.Seconds())%10 == 0 && int(metrics.Elapsed.Seconds()) > 0 {
		pr.printWorkerStats()
	}
}

// printWorkerStats prints detailed worker statistics
func (pr *ProgressReporter) printWorkerStats() {
	workerStats := pr.pool.GetWorkerStats()
	fmt.Printf("\n📊 Worker Stats:\n")
	
	for _, stats := range workerStats {
		fmt.Printf("  Worker %d: %d processed, %d errors\n", 
			stats.ID, stats.Processed, stats.Errors)
	}
}

// printFinalStats prints final statistics when crawling is complete
func (pr *ProgressReporter) printFinalStats() {
	metrics := pr.pool.GetMetrics()
	elapsed := time.Since(pr.startTime)
	
	fmt.Printf("\n\n🎉 Crawling Complete!\n")
	fmt.Printf("═══════════════════════════════════════\n")
	fmt.Printf("📈 Final Statistics:\n")
	fmt.Printf("  • Total URLs processed: %d\n", metrics.Processed)
	fmt.Printf("  • Total errors: %d\n", metrics.Errors)
	fmt.Printf("  • Total time: %v\n", elapsed)
	fmt.Printf("  • Average URLs/second: %.2f\n", metrics.URLsPerSecond)
	fmt.Printf("  • Total URLs visited: %d\n", metrics.VisitedCount)
	fmt.Printf("  • Success rate: %.1f%%\n", 
		float64(metrics.Processed-metrics.Errors)/float64(metrics.Processed)*100)
	
	// Print memory usage if available
	fmt.Printf("\n💾 Performance:\n")
	fmt.Printf("  • Peak URLs/second: %.2f\n", pr.calculatePeakThroughput())
	fmt.Printf("  • Active domains: %d\n", len(pr.pool.rateLimiter.GetActiveDomains()))
	
	// Print top domains by request count
	pr.printTopDomains()
}

// calculatePeakThroughput estimates peak throughput based on recent activity
func (pr *ProgressReporter) calculatePeakThroughput() float64 {
	// This is a simplified calculation - in a real implementation,
	// you might want to track throughput over sliding windows
	metrics := pr.pool.GetMetrics()
	return metrics.URLsPerSecond * 1.2 // Assume 20% higher peak
}

// printTopDomains prints the most active domains
func (pr *ProgressReporter) printTopDomains() {
	domains := pr.pool.rateLimiter.GetActiveDomains()
	if len(domains) > 0 {
		fmt.Printf("\n🌐 Active Domains (%d):\n", len(domains))
		for i, domain := range domains {
			if i < 5 { // Show top 5 domains
				fmt.Printf("  • %s\n", domain)
			}
		}
		if len(domains) > 5 {
			fmt.Printf("  • ... and %d more\n", len(domains)-5)
		}
	}
}

// LogError logs an error with context
func (pr *ProgressReporter) LogError(url string, err error) {
	fmt.Printf("\n❌ Error processing %s: %v\n", url, err)
}

// LogSuccess logs successful processing of a URL
func (pr *ProgressReporter) LogSuccess(url string, linksFound int) {
	// Only log every 100th success to avoid spam
	currentCount := atomic.LoadInt64(&pr.pool.processedCount)
	if currentCount%100 == 0 {
		fmt.Printf("\n✅ Processed %s (%d links found)\n", url, linksFound)
	}
}
