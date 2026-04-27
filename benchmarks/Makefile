.PHONY: help bench-latency bench-throughput cleanup

# Default target
help:
	@echo "Glambdar Benchmarking Tools"
	@echo ""
	@echo "Usage:"
	@echo "  make bench-latency      Run lifecycle benchmark (Cold/Warm start)"
	@echo "  make bench-throughput   Run throughput stress test with Apache Bench (ab)"
	@echo "  make cleanup            Stop test server and clean metadata"
	@echo ""
	@echo "Requirements:"
	@echo "  - Glambdar server running on :8000"
	@echo "  - Go installed"
	@echo "  - ab (Apache Bench) installed for throughput tests"

# ZIP the test function
function.zip: function/index.js
	cd ./function && zip -r ../function.zip .

# Run lifecycle benchmark
bench-latency: function.zip
	go run . -rateLimit 0

# Run throughput test (Requires ab)
bench-throughput: function.zip
	@echo "🚀 Starting High-Concurrency Stress Test (1,000 requests, 10 concurrent)..."
	@# Warm up first
	curl -s -X POST -F "file=@function.zip" http://localhost:8000/deploy > /dev/null
	curl -s -X POST -H "Content-Type: application/json" -d '{"body": "{}"}' http://localhost:8000/invoke/function > /dev/null
	@# Actual test
	@ab -n 1000 -c 10 -p <(echo '{"body": "{}"}') -T application/json http://localhost:8000/invoke/function | grep -E "Requests per second|Time per request"

# Cleanup
cleanup:
	curl -X DELETE http://localhost:8000/del/function > /dev/null
	rm -f function.zip
