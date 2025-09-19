# Demo Videos & Screenshots Directory

This directory contains demo videos and screenshots for the Go Concurrent Web Crawler README.

## Required Media Files

### 1. Live Crawling Demo Video
**File:** `screenshots/live-crawling-demo.mp4`

**Command to record:**
```bash
go run main.go -workers 10 -max 30 -progress 1s
```

**What to capture in the video:**
- **Startup banner** with configuration details
- **Live progress updates** showing:
  - URLs processed count increasing
  - Error count (should stay at 0)
  - URLs/second performance metric
  - Queue size and visited count
  - Active domains count
- **Worker statistics** (appears every 10 seconds)
- **Final summary** with comprehensive metrics
- **Total recording time**: 15-20 seconds is perfect

**Video quality tips:**
- Use high resolution (1920x1080 or higher)
- Ensure terminal text is clearly readable
- Record at least until you see worker stats appear
- Show the final summary at the end

### 2. JSON Output Screenshot
**File:** `screenshots/json-output.png`

**Command to run first:**
```bash
go run main.go -max 5 -workers 2
```

**What to capture:**
- Open `examples/output.json` in VS Code or a text editor
- Show the JSON structure with proper formatting
- Display 3-4 complete entries
- Highlight these key fields:
  - `"url"`: The crawled URL
  - `"title"`: Page title extracted
  - `"depth"`: BFS depth level
  - `"links"`: Array of found links
  - `"status"`: HTTP status code
  - `"processed_at"`: Timestamp
  - `"response_time_ms"`: Response time in nanoseconds

**Screenshot tips:**
- Use syntax highlighting if possible
- Show the array structure with multiple entries
- Make sure all field names are visible
- Crop to show relevant content only

## File Formats

- **Video**: MP4 format (most compatible)
- **Screenshot**: PNG format for best quality
- **Naming**: Use exact filenames specified above

## Recording Tools

### For Video Recording:
- **Windows**: Xbox Game Bar (Windows + G), OBS Studio, or Bandicam
- **macOS**: QuickTime Player, ScreenFlow, or OBS Studio
- **Linux**: OBS Studio, SimpleScreenRecorder, or Kazam

### For Screenshots:
- **Windows**: Snipping Tool, Windows + Shift + S
- **macOS**: Cmd + Shift + 4, or Screenshot app
- **Linux**: gnome-screenshot, or similar tools

## Why This Approach is Better

1. **Dynamic Demonstration**: Video shows real-time concurrency in action
2. **Performance Visualization**: Live metrics are more impressive than static numbers
3. **Professional Presentation**: Screen recordings are common in tech demos
4. **Interview Impact**: Shows the system working, not just code
5. **Efficiency**: Two focused demonstrations instead of six separate screenshots

## Alternative Commands

If the default command is too fast/slow, try:

```bash
# Slower for better video capture
go run main.go -workers 5 -max 20 -progress 2s

# Faster for quick demo
go run main.go -workers 15 -max 50 -progress 1s

# For JSON with more entries
go run main.go -max 10 -workers 3
```