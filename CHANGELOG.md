# Changelog

All notable changes to this project will be documented in this file.

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
