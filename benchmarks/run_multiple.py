import subprocess
import re
import statistics

def run_command(cmd, cwd):
    result = subprocess.run(cmd, shell=True, capture_output=True, text=True, cwd=cwd)
    return result.stdout

def parse_latency(output):
    cold = re.search(r"Cold Start Latency: ([\d\.]+)ms", output)
    warm = re.search(r"Average: ([\d\.]+)ms", output)
    return float(cold.group(1)) if cold else None, float(warm.group(1)) if warm else None

def parse_throughput(output):
    rps = re.search(r"Requests per second:\s+([\d\.]+)", output)
    return float(rps.group(1)) if rps else None

ITERATIONS = 3
results = {"cold": [], "warm": [], "rps": []}

print(f"Starting {ITERATIONS} benchmark iterations...")

for i in range(ITERATIONS):
    print(f"\n--- Iteration {i+1} ---")
    
    # Latency
    print("Measuring Latency...")
    lat_out = run_command("make bench-latency", "benchmarks")
    cold, warm = parse_latency(lat_out)
    if cold: results["cold"].append(cold)
    if warm: results["warm"].append(warm)
    print(f"Cold: {cold}ms, Warm: {warm}ms")
    
    # Throughput
    print("Measuring Throughput...")
    tp_out = run_command("make bench-throughput", "benchmarks")
    rps = parse_throughput(tp_out)
    if rps: results["rps"].append(rps)
    print(f"RPS: {rps}")

print("\n--- Final Results (Averaged) ---")
avg_cold = statistics.mean(results["cold"]) if results["cold"] else 0
avg_warm = statistics.mean(results["warm"]) if results["warm"] else 0
avg_rps = statistics.mean(results["rps"]) if results["rps"] else 0

print(f"Avg Cold Start: {avg_cold:.2f}ms")
print(f"Avg Warm Start: {avg_warm:.2f}ms")
print(f"Avg Throughput: {avg_rps:.2f} req/s")
