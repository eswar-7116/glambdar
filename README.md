# Glambdar

Glambdar is a minimal serverless function runtime written in Go for executing Node.js functions with Docker-based isolation.

It is simple and focuses on the core mechanics of a serverless runtime: deployment, invocation, isolation and IPC.

---

## Execution Flow

1. A function is uploaded as a zip file
2. The zip is extracted into a function-specific directory
3. On invocation:

   * A new Docker container is started
   * The function code is mounted
   * A Node.js worker executes the function
   * Communication between runtime and worker happens via Unix Domain Sockets (UDS)
4. The response is returned to the client
5. Metadata is tracked for each function
6. Functions can be queried or deleted via API routes

---

## Requirements

* **Docker**
* **Unix-based Environment** (Linux/macOS)
  > UDS is used for IPC, so Windows is not supported natively
* **Go** (for building the runtime)
* **Node.js** (inside Docker container, managed by the Node.js container image)

---

## Environment Setup

You **must** set the Glambdar base directory before running:

```bash
export GLAMBDAR_DIR="/absolute/path/to/glambdar"
```
> Add this to your shell’s RC file (e.g. `.bashrc`, `.zshrc`) to make it available in all new shell sessions.

* Glambdar relies on Docker for function isolation. Ensure the Docker daemon is running before starting the runtime.

---

## Quick Start

### 1. Clone the repository

```bash
git clone https://github.com/eswar-7116/glambdar.git
cd glambdar
```

### 2. Set the GLAMBDAR_DIR variable

```bash
export GLAMBDAR_DIR="$(pwd)"
```

### 3. Run the runtime

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

### 4. Deploy a function

```bash
curl -X POST \
  -F "file=@/path/to/myfunc.zip" \
  http://localhost:8000/deploy
```

> The function name is automatically inherited from the zip file name.

### 5. Invoke the function

```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{"name":"Glambdar"}' \
  http://localhost:8000/invoke/myfunc
```

### 6. List deployed functions

```bash
curl http://localhost:8000/info
```

### 7. Get function details

```bash
curl http://localhost:8000/info/myfunc
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

* Upload a zip file
* Glambdar extracts the zip file into `GLAMBDAR_DIR/functions/<name>`
* Initializes metadata

---

### Invoke a function

```
POST /invoke/:name
```

* Runs the function in an isolated Docker container
* One container per invocation
* Uses UDS for runtime-worker communication
> All invocations are **HTTP POST requests**.

---

### List all functions

```
GET /info
```

* Returns metadata for all deployed functions

---

### Get function details

```
GET /info/:name
```

* Returns metadata for a single function

---

### Delete a function

```
DELETE /del/:name
```

* Removes function code and metadata

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

* `req.headers`: request headers
* `req.body`: raw body string
* `await req.json()`: parsed JSON body

---

## Function Response Format

```js
{
  statusCode?: number,
  headers?: { [key: string]: string | string[] },
  body: any
}
```

* `statusCode` *(optional)* is the HTTP status code of the response (default: `200`)
* `headers` *(optional)* is the response headers
* `body` can be any JSON-serializable value
* Returned as the HTTP response body

---

## Testing

* **Unit tests** run by default
* **Integration tests** (Docker-dependent) are skipped unless enabled

Run only unit tests locally:

```bash
go test ./...
```

Run integration tests locally:

```bash
RUN_INTEGRATION_TESTS=1 go test ./...
```

---

## Design choices

* **Docker per invocation** for strong isolation
* **UDS over TCP** for low-latency IPC
* Simple IPC protocol (structured JSON)

---

**<p align="center">If you like this project, please consider giving this repo a star 🌟</p>**
