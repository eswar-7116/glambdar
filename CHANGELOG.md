# Changelog

All notable changes to this project will be documented in this file.

## [v4.0.0] - 2026-10-03

### Breaking Changes

- **Distributed Architecture**: Glambdar is no longer a single-process runtime. It is now split into two operational modes: `agent` (data plane) and `controller` (control plane); each launched as a subcommand. Running `glambdar` without a subcommand still starts the standalone single-node mode.
- **CLI migrated to Cobra**: All CLI flags now use hyphens (e.g., `--db-type`, `--grpc-port`) instead of underscores. Scripts using the old flag style must be updated.
- **Redis required for controller and clustered agent modes**: A running Redis instance is required when operating in `controller` or clustered `agent` modes.
- **SQLite dropped**: SQLite support has been fully removed. Only PostgreSQL and MySQL are supported as database backends.

### Features

- **Distributed Agent Mode** (`glambdar agent`): Worker agents now expose a gRPC server (`GlambdarAgent` service) to handle function invocations, preloads, evictions, and pool status reporting from the controller. Agents can optionally connect to a Redis cluster for state publishing.
- **Controller Mode** (`glambdar controller`): A new control plane mode that exposes an HTTP API and routes all invocations to worker agents via gRPC. The controller uses Redis to discover healthy nodes and maintain cluster state.
- **Protobuf & gRPC Definitions**: Added `proto/glambdar.proto` defining the `GlambdarAgent` service with `Invoke`, `PreloadFunction`, `EvictFunction`, and `GetStatus` RPCs.
- **gRPC Client Pool** (`internal/controller`): The controller maintains persistent, lazily-initialized gRPC connections to agent nodes with double-checked locking to prevent duplicate connections.
- **Redis Cluster State** (`internal/cluster`): New `StateProvider` interface backed by Redis for publishing and querying node health, warm pool status, and resource capacity. Agents publish heartbeats every 5 seconds.
- **Smart Cluster Routing** (`internal/cluster.Router`): Invocations are routed to the node with the most idle warm containers for the target function. On cache miss, routing falls back to the node with the highest available memory and CPU, using round-robin across ties.
- **Redis-Backed Rate Limiting**: In controller mode, rate limiting is enforced globally across all agents via a Redis token bucket (Lua script) instead of the local in-memory limiter used in standalone mode.
- **`EnsureCached` Helper Function**: Extracted a reusable `EnsureCached` function in `internal/functions` to decouple cache population (S3 download + extraction) from invocation, enabling agents to preload functions independently.
- **gVisor Container Sandboxing**: Function containers now run under the `runsc` (gVisor) OCI runtime with a host-UDS annotation, non-root user, dropped Linux capabilities, `no-new-privileges`, and a PID limit for improved security isolation.
- **Node Identity**: Each instance now carries a persistent UUID `node_id`, auto-generated and saved to `~/.glambdar/config.json` on first boot. The current node ID is exposed via the `GET /node-id` endpoint.
- **Batch Metadata Updates**: Invocation counters are now accumulated in memory and flushed to the database in batches, decoupling per-request metrics from synchronous DB writes to improve throughput.
- **S3 Function Storage**: Function zip files are stream-uploaded directly to an S3-compatible backend on deploy and downloaded on-demand for cache misses, replacing local-only disk storage.
- **Multipart Streaming Uploads**: Large function zip uploads are handled via multipart streaming to avoid buffering entire payloads in memory.
- **Config via Environment Variables and CLI Flags**: All configuration fields can now be set via `GLMBD_*` environment variables or Cobra CLI flags, with precedence: CLI flags > env vars > `config.json`.

### Refactoring & Improvements

- **Scalable Docker API Interface**: `DockerAPI` and `MockDockerAPI` interfaces were redesigned to be more scalable and maintainable, making it easier to add new Docker operations without widespread refactoring.
- **`CopyToContainer` for Worker Script Injection**: Replaced bind-mounts with `CopyToContainer` to avoid Docker-in-Docker (DinD) and Docker-out-of-Docker (DooD) permission issues in CI and containerized environments.
- **Cobra CLI Framework**: Migrated the CLI from manual `flag` package parsing to Cobra, resolving flag-parsing edge cases across subcommands and enabling subcommand-local flags.
- **Config field `db_config` renamed to `config`**: Internal config package organization streamlined.
- **Subcommand-Based Mode Detection**: Mode (standalone vs. agent vs. controller) is now determined by the subcommand used to launch Glambdar, not a user-defined config field, eliminating ambiguous startup behavior.
- **Deletion Cleanup**: Function deletion now correctly removes S3 objects and local cache directories in all cases.
- **Container File Ownership**: Fixed UID issues when copying files into containers under non-root users with gVisor.

### Bug Fixes

- **CI Integration Tests**: Fixed missing directory issues and updated CI to perform complete integration tests end-to-end, including Redis and PostgreSQL service containers.

### Testing

- **Unit Tests**: Added unit tests for the controller mode, gRPC client pool, Redis cluster state, Redis rate limiter, and `EnsureCached` cache/storage download paths.
- **Integration Tests**: Added end-to-end integration tests for the distributed architecture covering agent gRPC handlers and cluster routing.
- **CI Updated**: GitHub Actions workflow now runs on both `main` and `dev` branches, spins up Redis alongside PostgreSQL, installs gVisor (`runsc`) in the CI environment, and runs the full test suite with `-race -count=1`.

---

## [v3.0.0] - 2026-07-15

### Breaking Changes

- **Authentication & RBAC**: All API endpoints (except `/health`) are now protected by an `X-API-Key` header. Requests without a valid key will return `401 Unauthorized`.
- **Role-Based Access Control**: Keys are now assigned discrete roles (`admin`, `deployer`, `invoker`, `viewer`), strictly limiting endpoint access based on a default-deny policy.

### Features

- **Database Support**: Added support for PostgreSQL and MySQL alongside SQLite. The database connection can be configured via `~/.glambdar/config.json`.
- **Async Audit Logging**: Administrative and key-based actions are now continuously audited to the database asynchronously, maintaining warm-invocation latency goals.
- **Key Management API**: New keys can be generated, promoted, or revoked dynamically via the `/auth/keys` endpoints without downtime.
- **Bun Runtime Migration**: Migrated the worker script from Node.js to Bun to optimize warm-start execution paths.
- **Native UDS Server**: Replaced Node.js `net.createServer` with Bun's native `Bun.listen`, bypassing stream abstraction overhead.
- **ESM Native Resolving**: Replaced synchronous Node.js `require()` with dynamic `await import()`, natively resolving CJS and ESM code.
- **Graceful Shutdown**: Utilizes `server.stop(true)` to gracefully drain and close active socket connections.
- **Updated Base Image**: Migrated default container runtime image from `node:25-slim` to `oven/bun:slim`.

### Bug Fixes

- **Concurrent Pool Access**: Resolved a data race on `Entry.LastUsed` during concurrent `Release` calls by introducing a mutex lock on `Entry`.
- **CI/CD Testing**: Added `-race` and `-count=1` flags to both the unit test and integration test workflows to ensure concurrent race conditions are automatically detected in the pipeline.

### Performance

- **Warm Start Latency**: Improved to **~0.99 ms** (from ~1.06 ms, **7% faster**).
- **Warm Throughput**: Increased to **~2,449 req/s** (from ~2,248 req/s under identical test conditions).

---

## [v2.0.0] - 2026-04-27

### Features

- **EWMA-Based Predictive Pre-Warming**: Added a traffic-aware container pre-warmer that uses Exponentially Weighted Moving Average (EWMA) with dynamic alpha to predict demand and proactively spin up idle containers, eliminating cold starts under burst loads.
- **Traffic Prediction Engine**: New `internal/ewma` package with a `TrafficPredictor` that dynamically adjusts its smoothing factor based on traffic deviation for responsive scaling.
- **Per-Pool Invoke Tracking**: Each container pool now tracks invocation counts via an atomic counter, feeding the EWMA predictor every 30 seconds.
- **Multi-Method Support**: Functions now support `GET`, `POST`, `PUT`, `PATCH`, and `DELETE` methods for invocations. The request method is passed to the function handler via `req.method`.
- **Automatic Pool Cleanup**: Warm container pools are now immediately drained and containers are stopped when a function is deleted.
- **Add Benchmarking Scripts**: Added benchmarking scripts used to analyze the engine's performance.
- **Concurrent Request Handling**: A single container instance can now handle multiple concurrent requests (up to a configurable limit), improving resource utilization and drastically reducing cold starts by 42%.
- **Smart Pooling Logic**: Updated the pool manager to keep containers in the "Idle" pool until they reach their concurrency threshold, allowing for better "saturation" routing.
- **Monitoring Improvements**: Added `ColdStart` field to invocation responses to track and benchmark container creation events.
- **Per-Function Rate Limiting**: Added support for setting a maximum requests per second limit during function deployment.
- **Dynamic Configuration**: New `POST /config/:name` endpoint allowing real-time updates to function rate limits without redeploying.
- **Intelligent Burst Scaling**: Implemented a burst logic that scales at 10% of the rate limit to provide better throughput management.
- **Persistent Workers**: Transitioned to persistent Unix Domain Socket (UDS) servers for workers, significantly improving invocation performance.
- **Auto-Scaling & Latency**: Achieved a **99.6% reduction in latency** through auto-scaling container pools and persistent UDS workers.
- **Log Management**: Functions now support log extraction and storage in a centralized SQLite database.
- **Container Pooling**: Implemented per-function container pooling for optimized resource reuse.
- **Cron Jobs**: Integrated cron jobs to automatically remove stale containers.
- **Docker Integration**: Added automatic image pulling and Docker reachability guards.
- **Storage Migration**: Migrated metadata storage from JSON files to a centralized SQLite database to prevent race conditions.
- **Directory Structure**: Simplified storage by using a `.glambdar` directory for worker scripts and deployed functions.

### Refactoring & Improvements

- **Benchmarking Suite Rewrite**: Replaced the monolithic `benchmark_cli.go` with a modular benchmarking suite (`benchmark.go`, `client.go`, `main.go`, `types.go`) for clearer separation of concerns.
- **Shared Socket Utility**: Extracted `waitForSocket` into a standalone `internal/sockutil` package to allow reuse by both the invoke path and the pre-warmer without import cycles.
- **`GetOrCreate` Error Handling**: `PoolManager.GetOrCreate` now returns an error, enabling graceful handling of predictor initialization failures.
- **Docker SDK Migration**: Transitioned from using Docker CLI via `exec.Command` to the official Docker SDK for more robust container management.
- Centralized configuration management in `internal/config`.
- Reorganized project structure by moving the entry point to the root.
- Added comprehensive unit tests for logging functionality.
- Exported `BaseDir` path for better internal visibility.

### Documentation & Tests

- Updated README with latest benchmark results and design choices referencing IEEE methodologies.
- Updated README with new API documentation and latest benchmark results.
- Added comprehensive unit and integration tests for rate limiting and configuration management.
- **CI/CD**: Added integration test steps to the GitHub Actions workflow to ensure multi-method and pool management stability.

### Bug Fixes

- **Resource Management**: Fixed a temporary file leak in the deployment process where zip files were not cleaned up from `/tmp`.
- **Handler Robustness**: Improved the standard test function to handle null headers and gracefully echo request metadata.
- **API Reliability**: Corrected response status handling and updated internal versioning variables.
- Fixed several race conditions in the container pool acquisition logic.
- Ensured `MaxConcurrency` settings are safely handled as `int32`.
- Fixed Docker client lifecycle management to ensure closure after server shutdown.
- Ensured `functions/` directory is created if it does not exist.
- Improved error messaging across the system.
- Added `.gitignore` patterns for build outputs.

### Performance

- **Cold Start Latency**: Reduced to **~230 ms** (from ~590 ms).
- **Warm Start Latency**: Improved to **~1.06 ms** (from ~1.3 ms).
- **Warm Throughput**: Increased to **~2,951 req/s** (from ~1,100 req/s).
- **Burst Cold Starts**: Reduced to **zero** across consecutive burst rounds thanks to predictive pre-warming.

---

## [v1.0.0] - 2026-04-10

Initial release with support for basic function deployment and invocation.
