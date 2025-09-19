package crawler

import (
	"sync"
)

// URL represents a URL with its depth for BFS traversal
type URL struct {
	URL   string
	Depth int
}

// URLQueue represents a thread-safe queue for managing URLs to be processed.
// Uses a buffered channel for efficient concurrent access and a mutex-protected
// map for O(1) visited URL lookups to prevent revisiting URLs.
type URLQueue struct {
	queue   chan URL
	visited map[string]bool
	mu      sync.RWMutex // Use RWMutex for better read performance
	closed  bool
}

// NewURLQueue initializes a new URLQueue with a specified buffer size.
// The buffered channel allows up to bufferSize URLs to be queued without blocking.
func NewURLQueue(bufferSize int) *URLQueue {
	return &URLQueue{
		queue:   make(chan URL, bufferSize),
		visited: make(map[string]bool),
		closed:  false,
	}
}

// NewURL creates a new URL struct with the given URL and depth
func NewURL(url string, depth int) URL {
	return URL{URL: url, Depth: depth}
}

// Enqueue adds a new URL to the queue if it hasn't been visited yet and depth is within limits.
// Returns true if the URL was successfully enqueued, false if already visited or queue is closed.
func (q *URLQueue) Enqueue(url URL) bool {
	// Use write lock for updating visited map and enqueueing
	q.mu.Lock()
	defer q.mu.Unlock()

	// Check if queue is closed
	if q.closed {
		return false
	}

	// Check if URL has already been visited
	if q.visited[url.URL] {
		return false
	}

	q.visited[url.URL] = true

	// Non-blocking enqueue to prevent deadlock
	select {
	case q.queue <- url:
		return true
	default:
		// If channel is full, return false
		return false
	}
}

// Dequeue retrieves a URL from the queue.
// Returns the URL and a boolean indicating if the queue is still open.
func (q *URLQueue) Dequeue() (URL, bool) {
	url, ok := <-q.queue
	return url, ok
}

// IsEmpty checks if the queue is empty.
// Note: This is a best-effort check due to concurrent access.
func (q *URLQueue) IsEmpty() bool {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.queue) == 0
}

// Size returns the current number of URLs in the queue.
func (q *URLQueue) Size() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.queue)
}

// VisitedCount returns the number of URLs that have been visited.
func (q *URLQueue) VisitedCount() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.visited)
}

// Close closes the queue channel and marks it as closed.
// This signals to workers that no more URLs will be added.
func (q *URLQueue) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()
	if !q.closed {
		close(q.queue)
		q.closed = true
	}
}

// isClosed checks if the queue is closed (thread-safe).
func (q *URLQueue) isClosed() bool {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.closed
}

// IsVisited checks if a URL has already been visited (thread-safe).
func (q *URLQueue) IsVisited(url string) bool {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.visited[url]
}