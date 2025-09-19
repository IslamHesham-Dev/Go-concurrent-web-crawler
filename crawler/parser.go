package crawler

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// CrawlResult represents the result of crawling a single URL
type CrawlResult struct {
	URL         string    `json:"url"`
	Title       string    `json:"title"`
	Depth       int       `json:"depth"`
	Links       []string  `json:"links"`
	Status      int       `json:"status"`
	Error       string    `json:"error,omitempty"`
	ProcessedAt time.Time `json:"processed_at"`
	ResponseTime time.Duration `json:"response_time_ms"`
}

// HTTPClient is a configurable HTTP client with timeout and retry logic
type HTTPClient struct {
	client  *http.Client
	retries int
}

// NewHTTPClient creates a new HTTP client with configurable timeout and retries
func NewHTTPClient(timeout time.Duration, retries int) *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		retries: retries,
	}
}

// FetchURL fetches a URL with exponential backoff retry logic
func (c *HTTPClient) FetchURL(rawURL string) (*http.Response, error) {
	var lastErr error
	
	for attempt := 0; attempt <= c.retries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s, 8s...
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			time.Sleep(backoff)
		}

		resp, err := c.client.Get(rawURL)
		if err != nil {
			lastErr = err
			continue
		}

		// Check for successful status codes
		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			return resp, nil
		}

		resp.Body.Close()
		
		// Don't retry on client errors (4xx) except for 429 (rate limited)
		if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != 429 {
			return nil, fmt.Errorf("client error: %d %s", resp.StatusCode, resp.Status)
		}

		lastErr = fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	return nil, fmt.Errorf("failed after %d attempts: %v", c.retries+1, lastErr)
}

// ParseHTML parses HTML content and extracts title and links
func ParseHTML(resp *http.Response) (string, []string, error) {
	defer resp.Body.Close()

	// Parse the HTML document
	doc, err := html.Parse(resp.Body)
	if err != nil {
		return "", nil, fmt.Errorf("failed to parse HTML: %v", err)
	}

	// Extract title
	title := extractTitle(doc)
	if title == "" {
		title = "No Title"
	}

	// Extract links
	baseURL := resp.Request.URL.String()
	links := extractLinks(doc, baseURL)

	return title, links, nil
}

// extractTitle recursively searches for the title element in the HTML document
func extractTitle(n *html.Node) string {
	if n.Type == html.ElementNode && n.Data == "title" {
		if n.FirstChild != nil {
			return strings.TrimSpace(n.FirstChild.Data)
		}
		return ""
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if title := extractTitle(c); title != "" {
			return title
		}
	}
	return ""
}

// extractLinks recursively searches for anchor tags and extracts href attributes
func extractLinks(n *html.Node, baseURL string) []string {
	var links []string
	
	if n.Type == html.ElementNode && n.Data == "a" {
		for _, attr := range n.Attr {
			if attr.Key == "href" {
				link := normalizeURL(attr.Val, baseURL)
				if link != "" && isValidURL(link) {
					links = append(links, link)
				}
			}
		}
	}

	// Recursively process child nodes
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		links = append(links, extractLinks(c, baseURL)...)
	}

	return deduplicateLinks(links)
}

// normalizeURL converts relative URLs to absolute URLs
func normalizeURL(link, baseURL string) string {
	// Skip empty links, javascript:, mailto:, tel:, etc.
	if link == "" || strings.HasPrefix(link, "javascript:") || 
	   strings.HasPrefix(link, "mailto:") || strings.HasPrefix(link, "tel:") ||
	   strings.HasPrefix(link, "#") {
		return ""
	}

	// If it's already an absolute URL, return as is
	if strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://") {
		return link
	}

	// Parse the base URL
	base, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}

	// Parse the relative link
	relative, err := url.Parse(link)
	if err != nil {
		return ""
	}

	// Resolve the relative URL against the base URL
	resolved := base.ResolveReference(relative)
	return resolved.String()
}

// isValidURL checks if a URL is valid and should be crawled
func isValidURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	// Only crawl HTTP and HTTPS URLs
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}

	// Skip URLs without a host
	if parsed.Host == "" {
		return false
	}

	// Skip common file extensions that are not HTML
	path := strings.ToLower(parsed.Path)
	skipExtensions := []string{".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx",
		".zip", ".rar", ".tar", ".gz", ".jpg", ".jpeg", ".png", ".gif", ".svg", ".ico",
		".css", ".js", ".xml", ".json", ".txt", ".csv"}

	for _, ext := range skipExtensions {
		if strings.HasSuffix(path, ext) {
			return false
		}
	}

	return true
}

// deduplicateLinks removes duplicate links from the slice
func deduplicateLinks(links []string) []string {
	seen := make(map[string]bool)
	var unique []string

	for _, link := range links {
		if !seen[link] {
			seen[link] = true
			unique = append(unique, link)
		}
	}

	return unique
}

// IsSameDomain checks if two URLs belong to the same domain
func IsSameDomain(url1, url2 string) bool {
	domain1, err1 := ExtractDomain(url1)
	domain2, err2 := ExtractDomain(url2)
	
	if err1 != nil || err2 != nil {
		return false
	}
	
	return domain1 == domain2
}