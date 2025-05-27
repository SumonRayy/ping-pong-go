# Ping-Pong-Go

A health check service that implements a ping-pong mechanism between services. It can be used to monitor the health of distributed systems by having services ping each other at configurable intervals.

## Features

- Configurable ping intervals
- Custom headers for ping requests
- Health check endpoints
- Consecutive failure tracking
- Graceful shutdown on repeated failures
- Colored logging output
- Environment variable and flag-based configuration

## Installation

### As a Library

```bash
go get github.com/SumonRayy/ping-pong-go
```

### As a CLI Tool

```bash
# Install the CLI tool
go install github.com/SumonRayy/ping-pong-go/cmd/pingpong@latest

# Or clone and build from source
git clone https://github.com/SumonRayy/ping-pong-go.git
cd ping-pong-go
go install ./cmd/pingpong
```

## Usage

### As a Library

```go
package main

import (
    "context"
    "time"
    
    "github.com/SumonRayy/ping-pong-go/pkg/pingpong"
)

func main() {
    config := pingpong.Config{
        ServerURL:           "http://example.com/health",
        OwnURL:              "http://localhost:8080/health",
        PingInterval:        2 * time.Second,
        MaxConsecutiveFails: 3,
        MaxRetries:          3,
    }

    service := pingpong.NewService(config)
    ctx := context.Background()
    
    if err := service.Start(ctx); err != nil {
        panic(err)
    }
    
    // ... your application code ...
    
    service.Stop()
}
```

### As a CLI Tool

```bash
# Basic usage
pingpong

# With custom configuration
pingpong --server-url="http://example.com/health" \
         --ping-interval="5000" \
         --own-url="http://localhost:8080/health" \
         --max-retries="5" \
         --max-consecutive-fails="3"
```

### Environment Variables

You can configure the service using environment variables:

- `SERVER_URL`: URL of the server to ping (default: "http://localhost:8081/health")
- `OWN_URL`: URL of your own health check endpoint (default: "http://localhost:8080/health")
- `PING_INTERVAL`: Ping interval in milliseconds (default: 2000)
- `MAX_RETRIES`: Maximum number of retries for each ping (default: 3)
- `MAX_CONSECUTIVE_FAILS`: Maximum number of consecutive failures before shutdown (default: 3)

### Command-line Flags

- `--server-url`: Server URL to ping
- `--ping-interval`: Ping interval in milliseconds
- `--own-url`: Own health check URL
- `--max-retries`: Maximum number of retries
- `--max-consecutive-fails`: Maximum number of consecutive failures before shutdown

## Project Structure

```
.
├── cmd/
│   └── pingpong/          # CLI application
│       └── main.go
├── pkg/
│   └── pingpong/          # Library package
│       ├── pingpong.go    # Main package code
│       └── pingpong_test.go
├── go.mod
├── go.sum
└── README.md
```

## Health Check

The service exposes a health check endpoint at `/health`. It returns:
- `200 OK` if the service is healthy (last successful ping within 15 minutes)
- `503 Service Unavailable` if the service is unhealthy

## Testing

Run the tests using:

```bash
go test ./...
```

## Docker Support

Build and run using Docker:

```bash
docker build -t ping-pong-go .
docker run -p 8080:8080 ping-pong-go
```

## License

MIT License - see [LICENSE](LICENSE) file for details

## Author

[SumonRayy](https://sumonrayy.xyz)

## 🙏 Thanks for checking out the project!
### ⭐ Give it a star if you like it!

Follow me:

[![GitHub](https://img.shields.io/badge/GitHub-100000?style=for-the-badge&logo=github&logoColor=white)](https://github.com/SumonRayy/)
[![Buy Me A Coffee](https://img.shields.io/badge/Buy_Me_A_Coffee-FFDD00?style=for-the-badge&logo=buy-me-a-coffee&logoColor=black)](https://www.buymeacoffee.com/sumonrayyy)
[![Website](https://img.shields.io/badge/Website-4285F4?style=for-the-badge&logo=google-chrome&logoColor=white)](https://sumonrayy.xyz/)