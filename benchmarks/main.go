package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

const (
	defaultBaseURL = "http://localhost:8000"
	funcName       = "function"
	zipPath        = "function.zip"
	warmRuns       = 100
	burstSize      = 20
	prewarmWait    = 35 * time.Second
)

func main() {
	rateLimit := flag.Int("rateLimit", 0, "Rate limit for the function (0 for unlimited)")
	flag.Parse()

	if err := ensureZip(); err != nil {
		fmt.Printf("Failed to find zip: %v\n", err)
		os.Exit(1)
	}

	client := NewClient(defaultBaseURL)

	fmt.Printf("Starting Glambdar Benchmark (Rate Limit: %d)...\n\n", *rateLimit)

	// 0. Cleanup
	client.Cleanup(funcName)

	// 1. Deploy
	fmt.Println("Deploying function...")
	if err := client.Deploy(zipPath, *rateLimit); err != nil {
		fmt.Printf("Deployment failed: %v\n", err)
		os.Exit(1)
	}

	// 2. Cold Start
	fmt.Println("Measuring Cold Start...")
	coldLat, _ := client.MeasureInvoke(funcName)
	fmt.Printf("Cold Start Latency: %v\n\n", coldLat)

	// 3. Warm Starts
	fmt.Printf("Measuring %d Warm Starts...\n", warmRuns)
	var total time.Duration
	var min, max time.Duration
	successful, rate429 := 0, 0

	for range warmRuns {
		lat, status := client.MeasureInvoke(funcName)
		if status == 429 {
			rate429++
			continue
		}
		if status != 200 {
			fmt.Printf("Unexpected status: %d\n", status)
			continue
		}
		successful++
		total += lat
		if successful == 1 || lat < min {
			min = lat
		}
		if lat > max {
			max = lat
		}
	}

	if successful > 0 {
		fmt.Printf("Warm Start avg: %v  min: %v  max: %v  (%d/%d successful)\n",
			total/time.Duration(successful), min, max, successful, warmRuns)
	}
	if rate429 > 0 {
		fmt.Printf("Rate limited: %d\n", rate429)
	}

	// 3. Predictive Pre-warming
	fmt.Printf("\nMeasuring Predictive Pre-warming (burst=%d, wait=%v)...\n", burstSize, prewarmWait)
	c1, w1 := RunConcurrentBurst(client, funcName, burstSize)
	fmt.Printf("Burst 1 — cold: %d  warm: %d\n", c1, w1)
	fmt.Printf("Waiting %v for prewarmer ticker...\n", prewarmWait)
	time.Sleep(prewarmWait)
	c2, w2 := RunConcurrentBurst(client, funcName, burstSize)
	fmt.Printf("Burst 2 — cold: %d  warm: %d\n", c2, w2)
	fmt.Printf("Cold start reduction: %d -> %d\n", c1, c2)
}

func ensureZip() error {
	_, err := os.Stat(zipPath)
	return err
}
