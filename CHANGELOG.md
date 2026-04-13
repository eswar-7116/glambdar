# Changelog

All notable changes to this project will be documented in this file.

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
