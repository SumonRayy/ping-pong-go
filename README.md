# Ping Pong Service

## Overview

The Ping Pong Service is a Go application designed to:
- Periodically ping a specified server.
- Log the status of the server pings.
- Call its own `/health` endpoint after every successful ping to monitor its internal health.

This service is configured via environment variables, supports `.env` files, and includes a health check API for external monitoring.

---

## Features

- Periodically sends HTTP `GET` requests to a target server.
- Logs the status of each ping, including successes and failures.
- Exposes a `/health` endpoint to report its health status.
- Automatically calls its own health endpoint after successful pings.
- Fully configurable via environment variables or `.env` files.

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

### Example `.env` File

```dotenv
SERVER_URL=https://example.com
OWN_URL=http://localhost:8080/health
PING_INTERVAL=60000
PORT=8080
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

Logs are output to the console.

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