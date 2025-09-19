#!/bin/bash

# Go Concurrent Web Crawler - Example Usage Script
# This script demonstrates various ways to run the crawler

echo "🕷️  Go Concurrent Web Crawler - Example Usage"
echo "=============================================="

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.21 or later."
    exit 1
fi

# Check Go version
GO_VERSION=$(go version | cut -d' ' -f3 | sed 's/go//')
echo "✅ Go version: $GO_VERSION"

# Install dependencies
echo "📦 Installing dependencies..."
go mod tidy

# Example 1: Basic usage with default settings
echo ""
echo "🚀 Example 1: Basic usage with default settings"
echo "Command: go run main.go"
echo "This will crawl https://httpbin.org with 10 workers, depth 3"
echo "Press Enter to continue..."
read

go run main.go

# Example 2: Custom configuration
echo ""
echo "🚀 Example 2: Custom configuration"
echo "Command: go run main.go -url https://httpbin.org -workers 5 -depth 2 -rate 5"
echo "This will crawl with 5 workers, depth 2, and 5 req/sec rate limit"
echo "Press Enter to continue..."
read

go run main.go -url "https://httpbin.org" -workers 5 -depth 2 -rate 5 -max 20

# Example 3: With race detection
echo ""
echo "🚀 Example 3: Running with race detection"
echo "Command: go run -race main.go -max 10"
echo "This will check for race conditions while crawling"
echo "Press Enter to continue..."
read

go run -race main.go -max 10

# Example 4: Performance testing
echo ""
echo "🚀 Example 4: Running benchmarks"
echo "Command: go test -bench=. -benchmem ./crawler/"
echo "This will run performance benchmarks"
echo "Press Enter to continue..."
read

go test -bench=. -benchmem ./crawler/

echo ""
echo "✅ All examples completed!"
echo "📁 Check examples/output.json for the crawl results"
echo "📊 Review the console output for performance metrics"
