# Ping-Pong-Go Service

A robust health check service that pings a specified server and maintains its own health status.

## Features

- Configurable ping interval
- Custom headers support
- Automatic retry mechanism
- Graceful shutdown
- Health check endpoint
- Local test server
- Environment variable configuration
- Command-line flag support
- Configurable max consecutive failures
- Colorful logging with timestamps

## Installation

```bash
go get github.com/yourusername/ping-pong
```

## Configuration

The service can be configured using environment variables or command-line flags.

### Environment Variables

- `SERVER_URL`: URL of the server to ping (default: "http://localhost:8081/health")
- `PING_INTERVAL`: Interval between pings in milliseconds (default: "2000")
- `OWN_URL`: URL of the service's own health check endpoint (default: "http://localhost:8080/health")
- `MAX_RETRIES`: Maximum number of retries for each ping attempt (default: "3")
- `MAX_CONSECUTIVE_FAILS`: Maximum number of consecutive failures before shutdown (default: "3")

### Command-line Flags

- `-local`: Start a local test server
- `-server-url`: Server URL to ping
- `-ping-interval`: Ping interval in milliseconds
- `-own-url`: Own health check URL
- `-max-retries`: Maximum number of retries
- `-max-consecutive-fails`: Maximum number of consecutive failures before shutdown

## Usage

### Basic Usage

```bash
./ping-pong
```

### With Environment Variables

```bash
export SERVER_URL="http://example.com/health"
export PING_INTERVAL="5000"
export OWN_URL="http://localhost:8080/health"
export MAX_RETRIES="5"
export MAX_CONSECUTIVE_FAILS="3"
./ping-pong
```

### With Command-line Flags

```bash
./ping-pong -server-url "http://example.com/health" -ping-interval "5000" -own-url "http://localhost:8080/health" -max-retries 5 -max-consecutive-fails 3
```

### Start Local Test Server

```bash
./ping-pong -local
```

## Health Check

The service exposes a health check endpoint at `/health`. It returns:
- `200 OK` if the service is healthy (last successful ping within 15 minutes)
- `503 Service Unavailable` if the service is unhealthy

## Automatic Shutdown

The service will automatically shut down in the following scenarios:
1. After reaching the maximum number of consecutive failures (configurable via `MAX_CONSECUTIVE_FAILS` or `-max-consecutive-fails`)
2. When receiving a manual interrupt signal (Ctrl+C)

## Testing

Run the tests using:

```bash
go test -v
```

## Docker Support

Build and run using Docker:

```bash
docker build -t ping-pong .
docker run -p 8080:8080 ping-pong
```

## License

MIT License - see LICENSE file for details

---

## Overview

The Ping Pong Service is a Go application designed to:
- Periodically ping a specified server.
- Log the status of the server pings.
- Call its own `/health` endpoint after every successful ping to monitor its internal health.

This service is configured via environment variables, supports `.env` files, and includes a health check API for external monitoring.

---

## Requirements

- Go 1.22.9+ installed on your system.
- A target server to ping.
- Environment variables configured for the service.

---

## Configuration

The service reads configuration from environment variables or a `.env` file. The following variables are required:

| Variable       | Description                                         | Example                     |
|----------------|-----------------------------------------------------|-----------------------------|
| `SERVER_URL`   | The URL of the server to ping.                      | `https://example.com`       |
| `OWN_URL`      | The URL of the service's own `/health` endpoint.    | `http://localhost:8080/health` |
| `PING_INTERVAL`| Interval between pings (in milliseconds).           | `60000`                     |
| `PORT`         | (Optional) Port for the health check server.        | `8080`                      |
| `MAX_RETRIES`  | Maximum number of retries for failed pings.         | `3`                         |

### Example `.env` File

```dotenv
SERVER_URL=https://example.com
OWN_URL=http://localhost:8080/health
PING_INTERVAL=60000
PORT=8080
MAX_RETRIES=3
```

---

## How It Works

1. **Ping Routine**:
   - The application periodically pings the `SERVER_URL` based on the `PING_INTERVAL`.
   - Logs whether the ping was successful or failed.

2. **Self-Monitoring**:
   - After every successful ping, the service calls its own `/health` endpoint to check its status.

3. **Health Endpoint**:
   - Available at `/health` (default: `http://localhost:8080/health`).
   - Reports `200 OK` if:
     - At least one successful ping has occurred.
     - The last successful ping was within the last 15 minutes.
   - Otherwise, returns `503 Service Unavailable`.

4. **Retry Mechanism**:
   - If a ping fails, the service will retry up to `MAX_RETRIES` times before giving up.

5. **Local Test Server**:
   - Run the service with the `-local` flag to start a local test server on port 8081.

6. **Command-Line Flags**:
   - Override environment variables using command-line flags:
     - `-server-url`: Server URL to ping
     - `-ping-interval`: Ping interval in milliseconds
     - `-own-url`: Own health check URL
     - `-max-retries`: Maximum number of retries

---

## Installation and Usage

### Clone the Repository

```bash
git clone https://github.com/your-repo/ping-pong-service.git
cd ping-pong-service
```

### Install Dependencies

Make sure you have Go installed. Then, install any necessary dependencies (like `godotenv`).

```bash
go mod tidy
```

### Run the Service

1. Set up environment variables or create a `.env` file.
2. Run the service:

```bash
go run main.go
```

### Test the Health Endpoint

You can test the health endpoint using `curl` or a browser:

```bash
curl http://localhost:8080/health
```

---

## Logging

The service logs:
- Ping attempts and their results.
- Calls to its own `/health` endpoint.
- Errors encountered during operations.

Logs are output to the console with timestamps and colors.

---

## Example Output

```plaintext
2024/11/19 12:00:00 Starting Ping-Pong Server...
2024/11/19 12:00:00 Health check endpoint available at /health
2024/11/19 12:00:00 Pinging server: https://example.com
2024/11/19 12:00:01 Ping successful! to server : https://example.com
2024/11/19 12:00:01 Calling own health check endpoint: http://localhost:8080/health
2024/11/19 12:00:01 Own health check successful!
```

---

## Notes

1. If no `.env` file is found, the service will fall back to environment variables.
2. Ensure the target server and the service's `/health` endpoint are accessible from the host running this application.
3. The health check endpoint is useful for integration with monitoring tools like Prometheus or uptime monitors.

---

## Future Enhancements

- Add support for custom headers or authentication for the ping requests.
- Implement retries for failed pings.
- Add more detailed health metrics.

---

## License

This project is licensed under the [MIT License](LICENSE).

---

## 🙏 Thanks for checking out the project!
### ⭐ Give it a star if you like it!

Follow me:

[![GitHub](https://img.shields.io/badge/GitHub-100000?style=for-the-badge&logo=github&logoColor=white)](https://github.com/SumonRayy/)
[![Buy Me A Coffee](https://img.shields.io/badge/Buy_Me_A_Coffee-FFDD00?style=for-the-badge&logo=buy-me-a-coffee&logoColor=black)](https://www.buymeacoffee.com/sumonrayyy)
[![Website](https://img.shields.io/badge/Website-4285F4?style=for-the-badge&logo=google-chrome&logoColor=white)](https://sumonrayy.xyz/)