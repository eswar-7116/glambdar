<p align="center">
  <img src="assets/logo.png" alt="Glambdar Logo" width="200">
</p>

# Glambdar

Glambdar is a minimal serverless function runtime written in Go for executing Node.js functions with Docker-based isolation.

It is simple and focuses on the core mechanics of a serverless runtime: deployment, invocation, isolation and IPC.

---

## Execution Flow

1. A function is uploaded as a zip file
2. The zip is extracted into a function-specific directory
3. On invocation:
   - A warm Docker container is acquired from the pool (or a new one started)
   - The function code is mounted
   - A Node.js worker executes the function
   - Communication between runtime and worker happens via Unix Domain Sockets (UDS)
   - After execution, the container is returned to the pool for reuse

4. The response is returned to the client
5. Metadata and execution logs are tracked in a SQLite database for each function
6. Functions can be queried or deleted via API routes

---

## Requirements

- **Docker**
- **Unix-based Environment** (Linux/macOS)
  > UDS is used for IPC, so Windows is not supported natively
- **Go** (for building the runtime)
- **Node.js** (inside Docker container, managed by the Node.js container image)

---

## Environment Setup

- Glambdar relies on Docker for function isolation. Ensure the Docker daemon is running before starting the runtime.

- The runtime will automatically create a `.glambdar` directory in your user home directory to store functions, logs, and metadata.

---

## Quick Start

### 1. Clone the repository

```bash
git clone https://github.com/eswar-7116/glambdar.git
cd glambdar
```

### 2. Run the runtime

#### Option A: Run directly (development)

```bash
go run ./cmd/glambdar
```

#### Option B: Build and run

```bash
go build -o glambdar ./cmd/glambdar
./glambdar
```

The runtime starts an HTTP server on **`localhost:8000`**.

### 3. Deploy a function

```bash
curl -X POST \
  -F "file=@/path/to/myfunc.zip" \
  http://localhost:8000/deploy
```

> The function name is automatically inherited from the zip file name.

### 4. Invoke the function

```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{"name":"Glambdar"}' \
  http://localhost:8000/invoke/myfunc
```

### 5. List deployed functions

```bash
curl http://localhost:8000/info
```

### 6. Get function details

```bash
curl http://localhost:8000/info/myfunc
```

### 7. Get function logs

```bash
curl http://localhost:8000/logs/myfunc
```

### 8. Delete a function

```bash
curl -X DELETE http://localhost:8000/del/myfunc
```

---

## API Routes

### Deploy a function

```
POST /deploy
```

- Upload a zip file
- Glambdar extracts the zip file into `GLAMBDAR_DIR/functions/<name>`
- Initializes metadata
- **Optional**: `rateLimit` (form field) - set a maximum requests per second for this function (default: `0` for unlimited)

### Configure a function

```
POST /config/:name
```

- **Body**: `{"rateLimit": number}`
- Updates the rate limit for a deployed function in real-time without redeploying.

### Invoke a function

```
POST /invoke/:name
```

- Runs the function in an isolated Docker container
- Uses a warm container pool for subsequent faster invocations
- Uses UDS for runtime-worker communication
  > All invocations are **HTTP POST requests**.

### Get function logs

```
GET /logs/:name
```

- Returns stdout and stderr execution logs for a single function

### List all functions

```
GET /info
```

- Returns metadata for all deployed functions

### Get function details

```
GET /info/:name
```

- Returns metadata for a single function

### Delete a function

```
DELETE /del/:name
```

- Removes function code, metadata, and logs

---

## Function Interface

Each deployed function **must** export a `handler` function from an `index.js` file.

### Requirements

- File name must be **`index.js`**
- The entry point must be **`exports.handler`**
- The handler must be an **async function**
- The handler receives the request object described below in the [Function Request Format](#function-request-format) section

### Example

```js
exports.handler = async (req) => {
  const jsonData = await req.json();

  return {
    statusCode: 200,
    body: {
      message: `Hello ${jsonData.name}!`,
    },
  };
};
```

If `handler` is missing or `index.js` is not present, the invocation will fail.

---

## Function Request Format

```js
{
  headers: { [key: string]: string | string[] },
  body: string,
  json(): Promise<any>
}
```

Inside the function:

- `req.headers`: request headers
- `req.body`: raw body string
- `await req.json()`: parsed JSON body

---

## Function Response Format

```js
{
  statusCode?: number,
  headers?: { [key: string]: string | string[] },
  body: any
}
```

- `statusCode` _(optional)_ is the HTTP status code of the response (default: `200`)
- `headers` _(optional)_ is the response headers
- `body` can be any JSON-serializable value
- Returned as the HTTP response body

---

## Testing

- **Unit tests** run by default
- **Integration tests** (Docker-dependent) are skipped unless enabled

Run only unit tests locally:

```bash
go test ./...
```

Run integration tests locally:

```bash
RUN_INTEGRATION_TESTS=1 go test ./...
```

---

## Performance Benchmarks

Glambdar is optimized for low-latency function execution using persistent per-function container pools, Unix Domain Socket (UDS) IPC, and EWMA-based predictive pre-warming.

| Metric                       | Result           |
| ---------------------------- | ---------------- |
| **Cold Start Latency**       | **~230 ms**      |
| **Warm Start Latency (Avg)** | **~1.06 ms**     |
| **Warm Throughput**          | **~2,951 req/s** |

**Benchmark Environment**

- Local Linux environment
- Intel i7 13th Gen, 16GB RAM
- Simple "ping" function returning static JSON
- Warm latency measured after worker/container initialization
- Throughput measured under concurrent warm load

---

## Design choices

- **Persistent Docker container pool** for reduced latency and auto-scaling
- **Intra-Function Concurrency:** Implemented a multi-request routing threshold (adapted from 2024 IEEE serverless optimization models) to drastically reduce cold starts under burst loads while maintaining strict process isolation.
- **EWMA-Based Predictive Pre-Warming:** Uses Exponentially Weighted Moving Average traffic prediction with dynamic alpha to proactively spin up containers before demand spikes, eliminating cold starts under burst loads.
- **UDS over TCP** for low-latency IPC
- Simple IPC protocol (structured JSON)

---

**<p align="center">If you like this project, please consider giving this repo a star 🌟</p>**
