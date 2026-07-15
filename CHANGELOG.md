# Changelog

All notable changes to this project will be documented in this file.

## [v4.0.0] - 2026-07-15

### Breaking Changes

- **Authentication & RBAC**: All API endpoints (except `/health`) are now protected by an `X-API-Key` header. Requests without a valid key will return `401 Unauthorized`.
- **Role-Based Access Control**: Keys are now assigned discrete roles (`admin`, `deployer`, `invoker`, `viewer`), strictly limiting endpoint access based on a default-deny policy.

### Features

- **Database Support**: Added support for PostgreSQL and MySQL alongside SQLite. The database connection can be configured via `~/.glambdar/db_config.json`.
- **Async Audit Logging**: Administrative and key-based actions are now continuously audited to the database asynchronously, maintaining warm-invocation latency goals.
- **Key Management API**: New keys can be generated, promoted, or revoked dynamically via the `/auth/keys` endpoints without downtime.

---

## [v3.0.1] - 2026-05-29

### Bug Fixes

- **Concurrent Pool Access**: Resolved a data race on `Entry.LastUsed` during concurrent `Release` calls by introducing a mutex lock on `Entry`.
- **CI/CD Testing**: Added `-race` and `-count=1` flags to both the unit test and integration test workflows to ensure concurrent race conditions are automatically detected in the pipeline.

## [v3.0.0] - 2026-05-25

### Features

- **Bun Runtime Migration**: Migrated the worker script from Node.js to Bun to optimize warm-start execution paths.
- **Native UDS Server**: Replaced Node.js `net.createServer` with Bun's native `Bun.listen`, bypassing stream abstraction overhead.
- **ESM Native Resolving**: Replaced synchronous Node.js `require()` with dynamic `await import()`, natively resolving CJS and ESM code.
- **Graceful Shutdown**: Utilizes `server.stop(true)` to gracefully drain and close active socket connections.
- **Updated Base Image**: Migrated default container runtime image from `node:25-slim` to `oven/bun:slim`.

### Performance

- **Warm Start Latency**: Improved to **~0.99 ms** (from ~1.06 ms, **7% faster**).
- **Warm Throughput**: Increased to **~2,449 req/s** (from ~2,248 req/s under identical test conditions).

---

## [v2.3.0] - 2026-04-27

### Features

- **EWMA-Based Predictive Pre-Warming**: Added a traffic-aware container pre-warmer that uses Exponentially Weighted Moving Average (EWMA) with dynamic alpha to predict demand and proactively spin up idle containers, eliminating cold starts under burst loads.
- **Traffic Prediction Engine**: New `internal/ewma` package with a `TrafficPredictor` that dynamically adjusts its smoothing factor based on traffic deviation for responsive scaling.
- **Per-Pool Invoke Tracking**: Each container pool now tracks invocation counts via an atomic counter, feeding the EWMA predictor every 30 seconds.

### Refactoring & Improvements

- **Benchmarking Suite Rewrite**: Replaced the monolithic `benchmark_cli.go` with a modular benchmarking suite (`benchmark.go`, `client.go`, `main.go`, `types.go`) for clearer separation of concerns.
- **Shared Socket Utility**: Extracted `waitForSocket` into a standalone `internal/sockutil` package to allow reuse by both the invoke path and the pre-warmer without import cycles.
- **`GetOrCreate` Error Handling**: `PoolManager.GetOrCreate` now returns an error, enabling graceful handling of predictor initialization failures.

### Performance

- **Cold Start Latency**: Reduced to **~230 ms** (from ~340 ms, **32% faster**).
- **Warm Start Latency**: Improved to **~1.06 ms** (from ~1.3 ms, **18% faster**).
- **Warm Throughput**: Increased to **~2,951 req/s** (from ~1,900 req/s, **55% higher**).
- **Burst Cold Starts**: Reduced to **zero** across consecutive burst rounds thanks to predictive pre-warming.

---

## [v2.2.1] - 2026-04-14

### Features

- **Multi-Method Support**: Functions now support `GET`, `POST`, `PUT`, `PATCH`, and `DELETE` methods for invocations. The request method is passed to the function handler via `req.method`.
- **Automatic Pool Cleanup**: Warm container pools are now immediately drained and containers are stopped when a function is deleted.
- **Add Benchmarking Scripts**: Added benchmarking scripts used to analyze the engine's performance.

### Bug Fixes & Improvements

- **Resource Management**: Fixed a temporary file leak in the deployment process where zip files were not cleaned up from `/tmp`.
- **Handler Robustness**: Improved the standard test function to handle null headers and gracefully echo request metadata.
- **CI/CD**: Added integration test steps to the GitHub Actions workflow to ensure multi-method and pool management stability.
- **API Reliability**: Corrected response status handling and updated internal versioning variables.

---

## [v2.2.0] - 2026-04-13

### Features

- **Concurrent Request Handling**: A single container instance can now handle multiple concurrent requests (up to a configurable limit), improving resource utilization and drastically reducing cold starts by 42%.
- **Smart Pooling Logic**: Updated the pool manager to keep containers in the "Idle" pool until they reach their concurrency threshold, allowing for better "saturation" routing.
- **Monitoring Improvements**: Added `ColdStart` field to invocation responses to track and benchmark container creation events.

### Performance

- **Cold Start Latency**: Reduced to **~340 ms** (from ~590 ms).
- **Warm Throughput**: Increased to **~1,900 req/s** (from ~1,100 req/s) due to concurrent execution.
- **Warm Start Latency**: Stable at **~1.3 ms**.

### Documentation & Bug Fixes

- Updated README with latest benchmark results and design choices referencing IEEE methodologies.
- Fixed several race conditions in the container pool acquisition logic.
- Ensured `MaxConcurrency` settings are safely handled as `int32`.

---

## [v2.1.0] - 2026-04-12

### Features

- **Per-Function Rate Limiting**: Added support for setting a maximum requests per second limit during function deployment.
- **Dynamic Configuration**: New `POST /config/:name` endpoint allowing real-time updates to function rate limits without redeploying.
- **Intelligent Burst Scaling**: Implemented a burst logic that scales at 10% of the rate limit to provide better throughput management.

### Performance

- **Warm Throughput**: Increased to **~1,100 req/s**.
- **Warm Start Latency**: Improved to **~1.3 ms**.

### Documentation & Tests

- Updated README with new API documentation and latest benchmark results.
- Added comprehensive unit and integration tests for rate limiting and configuration management.

---

## [v2.0.0] - 2026-04-11

### Features

- **Persistent Workers**: Transitioned to persistent Unix Domain Socket (UDS) servers for workers, significantly improving invocation performance.
- **Auto-Scaling & Latency**: Achieved a **99.6% reduction in latency** through auto-scaling container pools and persistent UDS workers.
- **Log Management**: Functions now support log extraction and storage in a centralized SQLite database.
- **Container Pooling**: Implemented per-function container pooling for optimized resource reuse.
- **Cron Jobs**: Integrated cron jobs to automatically remove stale containers.
- **Docker Integration**: Added automatic image pulling and Docker reachability guards.
- **Storage Migration**: Migrated metadata storage from JSON files to a centralized SQLite database to prevent race conditions.
- **Directory Structure**: Simplified storage by using a `.glambdar` directory for worker scripts and deployed functions.

### Refactoring & Improvements

- **Docker SDK Migration**: Transitioned from using Docker CLI via `exec.Command` to the official Docker SDK for more robust container management.
- Centralized configuration management in `internal/config`.
- Reorganized project structure by moving the entry point to the root.
- Added comprehensive unit tests for logging functionality.
- Exported `BaseDir` path for better internal visibility.

### Bug Fixes

- Fixed Docker client lifecycle management to ensure closure after server shutdown.
- Ensured `functions/` directory is created if it does not exist.
- Improved error messaging across the system.
- Added `.gitignore` patterns for build outputs.

---

## [v1.0.0] - 2026-04-10

Initial release with support for basic function deployment and invocation.
