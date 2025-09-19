# Go Concurrent Web Crawler - Example Usage Script (PowerShell)
# This script demonstrates various ways to run the crawler

Write-Host "🕷️  Go Concurrent Web Crawler - Example Usage" -ForegroundColor Green
Write-Host "==============================================" -ForegroundColor Green

# Check if Go is installed
try {
    $goVersion = go version
    Write-Host "✅ Go is installed: $goVersion" -ForegroundColor Green
} catch {
    Write-Host "❌ Go is not installed. Please install Go 1.21 or later." -ForegroundColor Red
    exit 1
}

# Install dependencies
Write-Host "📦 Installing dependencies..." -ForegroundColor Yellow
go mod tidy

# Example 1: Basic usage with default settings
Write-Host ""
Write-Host "🚀 Example 1: Basic usage with default settings" -ForegroundColor Cyan
Write-Host "Command: go run main.go" -ForegroundColor White
Write-Host "This will crawl https://httpbin.org with 10 workers, depth 3" -ForegroundColor Gray
Write-Host "Press Enter to continue..." -ForegroundColor Yellow
Read-Host

go run main.go

# Example 2: Custom configuration
Write-Host ""
Write-Host "🚀 Example 2: Custom configuration" -ForegroundColor Cyan
Write-Host "Command: go run main.go -url https://httpbin.org -workers 5 -depth 2 -rate 5" -ForegroundColor White
Write-Host "This will crawl with 5 workers, depth 2, and 5 req/sec rate limit" -ForegroundColor Gray
Write-Host "Press Enter to continue..." -ForegroundColor Yellow
Read-Host

go run main.go -url "https://httpbin.org" -workers 5 -depth 2 -rate 5 -max 20

# Example 3: With race detection
Write-Host ""
Write-Host "🚀 Example 3: Running with race detection" -ForegroundColor Cyan
Write-Host "Command: go run -race main.go -max 10" -ForegroundColor White
Write-Host "This will check for race conditions while crawling" -ForegroundColor Gray
Write-Host "Press Enter to continue..." -ForegroundColor Yellow
Read-Host

go run -race main.go -max 10

# Example 4: Performance testing
Write-Host ""
Write-Host "🚀 Example 4: Running benchmarks" -ForegroundColor Cyan
Write-Host "Command: go test -bench=. -benchmem ./crawler/" -ForegroundColor White
Write-Host "This will run performance benchmarks" -ForegroundColor Gray
Write-Host "Press Enter to continue..." -ForegroundColor Yellow
Read-Host

go test -bench=. -benchmem ./crawler/

Write-Host ""
Write-Host "✅ All examples completed!" -ForegroundColor Green
Write-Host "📁 Check examples/output.json for the crawl results" -ForegroundColor Yellow
Write-Host "📊 Review the console output for performance metrics" -ForegroundColor Yellow
