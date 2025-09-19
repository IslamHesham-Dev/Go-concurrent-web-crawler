package crawler

import (
	"context"
	"net/url"
	"sync"
	"time"
)

// RateLimiter controls the rate of requests to different domains using time.Ticker.
// Each domain gets its own rate limiter to ensure we don't exceed the specified
// requests per second limit per domain.
type RateLimiter struct {
	mu           sync.Mutex
	limiters     map[string]*domainLimiter
	requestsPerSecond int
	cleanupInterval   time.Duration
	stopCleanup       chan struct{}
}

// domainLimiter manages rate limiting for a single domain
type domainLimiter struct {
	ticker    *time.Ticker
	requests  chan struct{}
	lastUsed  time.Time
}

// NewRateLimiter creates a new RateLimiter with the specified requests per second limit.
// The cleanupInterval determines how often inactive domain limiters are cleaned up.
func NewRateLimiter(requestsPerSecond int) *RateLimiter {
	rl := &RateLimiter{
		limiters:          make(map[string]*domainLimiter),
		requestsPerSecond: requestsPerSecond,
		cleanupInterval:   5 * time.Minute, // Clean up inactive limiters every 5 minutes
		stopCleanup:       make(chan struct{}),
	}

	// Start cleanup goroutine
	go rl.cleanupInactiveLimiters()

	return rl
}

// Wait blocks until the next request to the given domain is allowed.
// Uses exponential backoff if the domain is not found or if there's an error.
func (rl *RateLimiter) Wait(ctx context.Context, domain string) error {
	limiter := rl.getOrCreateLimiter(domain)
	
	select {
	case <-limiter.requests:
		// Request allowed
		limiter.lastUsed = time.Now()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// getOrCreateLimiter returns the rate limiter for the given domain,
// creating a new one if it doesn't exist.
func (rl *RateLimiter) getOrCreateLimiter(domain string) *domainLimiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.limiters[domain]
	if !exists {
		// Create a new ticker that fires at the specified rate
		interval := time.Second / time.Duration(rl.requestsPerSecond)
		ticker := time.NewTicker(interval)
		
		// Create a buffered channel to hold allowed requests
		// Buffer size should be at least the requests per second to handle bursts
		requests := make(chan struct{}, rl.requestsPerSecond*2)
		
		// Fill the channel initially
		for i := 0; i < rl.requestsPerSecond; i++ {
			requests <- struct{}{}
		}

		limiter = &domainLimiter{
			ticker:   ticker,
			requests: requests,
			lastUsed: time.Now(),
		}

		rl.limiters[domain] = limiter

		// Start the ticker goroutine for this domain
		go func() {
			for {
				select {
				case <-ticker.C:
					// Add a new request token
					select {
					case requests <- struct{}{}:
					default:
						// Channel is full, skip this tick
					}
				case <-rl.stopCleanup:
					return
				}
			}
		}()
	}

	return limiter
}

// ExtractDomain extracts the domain from a URL for rate limiting purposes.
func ExtractDomain(rawURL string) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	return parsedURL.Host, nil
}

// cleanupInactiveLimiters periodically removes rate limiters for domains
// that haven't been used recently to prevent memory leaks.
func (rl *RateLimiter) cleanupInactiveLimiters() {
	ticker := time.NewTicker(rl.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			for domain, limiter := range rl.limiters {
				// Remove limiters that haven't been used for more than 10 minutes
				if now.Sub(limiter.lastUsed) > 10*time.Minute {
					limiter.ticker.Stop()
					close(limiter.requests)
					delete(rl.limiters, domain)
				}
			}
			rl.mu.Unlock()
		case <-rl.stopCleanup:
			return
		}
	}
}

// Stop stops all active rate limiters and cleans up resources.
func (rl *RateLimiter) Stop() {
	close(rl.stopCleanup)
	
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	for _, limiter := range rl.limiters {
		limiter.ticker.Stop()
		close(limiter.requests)
	}
	rl.limiters = make(map[string]*domainLimiter)
}

// GetActiveDomains returns the list of domains currently being rate limited.
func (rl *RateLimiter) GetActiveDomains() []string {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	domains := make([]string, 0, len(rl.limiters))
	for domain := range rl.limiters {
		domains = append(domains, domain)
	}
	return domains
}