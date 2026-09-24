<p align="center">
  <img src="assets/logo.png" alt="Glambdar Logo" width="200">
</p>

# Glambdar

Glambdar is a minimal serverless function runtime written in Go for executing Bun functions with Docker-based isolation.

It is simple and focuses on the core mechanics of a serverless runtime: deployment, invocation, isolation and IPC.

[![GoDoc](https://godoc.org/github.com/eswar-7116/glambdar?status.svg)](https://godoc.org/github.com/eswar-7116/glambdar)

---

## Execution Flow

1. A function is uploaded as a zip file via `/deploy`
2. The zip is stream-uploaded directly to S3 storage
3. On invocation:
   - If the function directory is not present locally in `~/.glambdar/functions`, the zip is downloaded from S3 storage and extracted on demand
   - A warm Docker container is acquired from the pool (or a new one started)
   - The function code is mounted
   - A Bun worker executes the function
   - Communication between runtime and worker happens via Unix Domain Sockets (UDS)
   - After execution, the container is returned to the pool for reuse

4. The response is returned to the client
5. Metadata and execution logs are tracked in a database for each function
6. Functions can be queried or deleted via API routes (deleting a function removes code from S3, local cache, metadata, and logs)

---

## Requirements

- **Docker**
- **gVisor (`runsc`)** (Container runtime sandbox for secure container isolation)
- **Unix-based Environment** (Linux/macOS)
  > UDS is used for IPC, so Windows is not supported natively
- **Go** (for building the runtime)
- **Bun** (inside Docker container, managed by the `oven/bun:slim` container image)
- **S3-compatible Object Storage** (AWS S3, SeaweedFS, MinIO, RustFS, Ceph, etc.)

---

## Environment Setup

- Glambdar relies on Docker and gVisor (`runsc`) for function isolation. Ensure the Docker daemon is running and `runsc` is registered as a Docker runtime before starting the runtime. Follow the [official gVisor installation guide](https://gvisor.dev/docs/user_guide/install/) to install `runsc` and configure Docker.

- Glambdar will automatically create a `.glambdar` directory in your user home directory for local function caches, config, and runtime metadata.

---

## Configuration

> **Configuration precedence:** CLI flags > Environment variables (`GLMBD_*`) > `config.json` values.
> Glambdar configuration can be customized by creating a `~/.glambdar/config.json` file.
> You can also configure via environment variables prefixed with `GLMBD_` or CLI flags (e.g., `--db_type`, `--dsn`). Run `glambdar --help` to see all available flags.

```jsonc
{
  "node_id": "", // Generated automatically and persisted for this instance
  "db_type": "postgres", // postgres or mysql
  "dsn": "postgres://user:pass@localhost:5432/glambdar",
  "s3": {
    "endpoint": "http://localhost:8333", // Custom endpoint for S3 compatible API (MinIO, SeaweedFS, etc.) or "" for AWS
    "region": "us-east-1",
    "bucket": "my-bucket",
    "access_key_id": "YOUR_ACCESS_KEY",
    "secret_access_key": "YOUR_SECRET_KEY",
    "session_token": "",
    "force_path_style": true, // Set to true when using custom S3 endpoints
  },
}
```

### Node Identity

Each Glambdar instance has a unique UUID `node_id`. If it is empty or missing when the configuration is first loaded, Glambdar generates a UUID and saves it to `~/.glambdar/config.json`. The ID remains stable across restarts and identifies the instance for distributed coordination.

The current node ID can be queried through the authenticated `/node-id` endpoint.

### Database

Glambdar requires a PostgreSQL or MySQL database. Set the `db_type` and `dsn` fields in `~/.glambdar/config.json` to point at your database instance.

_(Valid `db_type` values are `postgres` and `mysql`)_

### S3 Storage

Glambdar stores function zips in S3 storage instead of local disk storage.
Any S3-compatible provider is supported (e.g. AWS S3, SeaweedFS, MinIO, RustFS). The zips are downloaded and extracted locally on-demand when invoked.

### Authentication & RBAC

All API endpoints (except `/health`) require an API key passed via the `X-API-Key` header.
On the **very first boot**, Glambdar generates a Root Admin key and prints it to the console:

```text
Admin API key: glmbd_ak_xxxxxxxxxxxxxxxxxxxx
Save this key securely. It will NOT be shown again.
```

_(If lost, reset it using `glambdar reset-admin-key`)_

You can use the Root Admin key to generate secondary keys with restricted privileges via the `/auth/keys` endpoint.
The available roles are:

- `admin`: Full access to everything
- `deployer`: Can deploy, invoke, delete, and view functions/logs
- `invoker`: Can invoke functions and view basic info
- `viewer`: Read-only access to function info and logs

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
  -H "X-API-Key: glmbd_ak_YOUR_KEY_HERE" \
  -F "file=@/path/to/myfunc.zip" \
  http://localhost:8000/deploy
```

> The function name is automatically inherited from the zip file name.

### 4. Invoke the function

```bash
curl -X POST \
  -H "X-API-Key: glmbd_ak_YOUR_KEY_HERE" \
  -H "Content-Type: application/json" \
  -d '{"name":"Glambdar"}' \
  http://localhost:8000/invoke/myfunc
```

### 5. List deployed functions

```bash
curl -H "X-API-Key: glmbd_ak_YOUR_KEY_HERE" http://localhost:8000/info
```

### 6. Get function details

```bash
curl -H "X-API-Key: glmbd_ak_YOUR_KEY_HERE" http://localhost:8000/info/myfunc
```

### 7. Get function logs

```bash
curl -H "X-API-Key: glmbd_ak_YOUR_KEY_HERE" http://localhost:8000/logs/myfunc
```

### 8. Get the node ID

```bash
curl -H "X-API-Key: glmbd_ak_YOUR_KEY_HERE" http://localhost:8000/node-id
```

### 9. Delete a function

```bash
curl -X DELETE -H "X-API-Key: glmbd_ak_YOUR_KEY_HERE" http://localhost:8000/del/myfunc
```

---

## API Routes

### Deploy a function

```
POST /deploy
```

- Upload a zip file (`file` form field)
- Streamed directly to S3 storage (`<funcName>.zip`)
- Initializes metadata
- **Optional**: `funcName` (form field) - custom function name (defaults to zip filename without extension)
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
- Downloads function zip from S3 storage if not cached locally in `~/.glambdar/functions`
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

### Get node ID

```
GET /node-id
```

- Requires an API key with `info` permission
- Returns the persistent UUID assigned to this Glambdar instance

```json
{
  "nodeId": "<NODE_ID>"
}
```

### Delete a function

```
DELETE /del/:name
```

- Removes function code from S3 storage, local extracted directory, metadata, and logs

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
| **Warm Start Latency (Avg)** | **~0.99 ms**     |
| **Warm Throughput**          | **~2,449 req/s** |

**Benchmark Environment**

- Local Linux environment
- Intel i7 13th Gen, 16GB RAM
- Simple "ping" function returning static JSON
- Bun runtime (`oven/bun:slim`) with native UDS server (`Bun.listen`)
- Warm latency measured after worker/container initialization
- Throughput measured under concurrent warm load (1,000 requests, 10 concurrent)

---

## Design choices

- **Persistent Docker container pool** for reduced latency and auto-scaling
- **Intra-Function Concurrency:** Implemented a multi-request routing threshold (adapted from 2024 IEEE serverless optimization models) to drastically reduce cold starts under burst loads while maintaining strict process isolation.
- **EWMA-Based Predictive Pre-Warming:** Uses Exponentially Weighted Moving Average traffic prediction with dynamic alpha to proactively spin up containers before demand spikes, eliminating cold starts under burst loads.
- **UDS over TCP** for low-latency IPC
- Simple IPC protocol (structured JSON)

---

**<p align="center">If you like this project, please consider giving this repo a star 🌟</p>**
