package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

const (
	baseURL  = "http://localhost:8000"
	funcName = "function"
	zipPath  = "function.zip"
	warmRuns = 100
)

type InvokeRequest struct {
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

func main() {
	rateLimit := flag.Int("rateLimit", 0, "Rate limit for the function (0 for unlimited)")
	flag.Parse()

	// Ensure function is zipped
	if err := ensureZip(); err != nil {
		fmt.Printf("Failed to zip function: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Starting Glambdar Benchmark (Rate Limit: %d)...\n", *rateLimit)

	// 0. Cleanup
	cleanup()

	// 1. Deploy
	fmt.Println("Deploying function...")
	if err := deploy(*rateLimit); err != nil {
		fmt.Printf("Deployment failed: %v\n", err)
		os.Exit(1)
	}

	// 2. Cold Start
	fmt.Println("Measuring Cold Start...")
	coldStart, status := measureInvoke()
	fmt.Printf("Cold Start Latency: %v (Status: %d)\n", coldStart, status)

	// 3. Warm Starts
	fmt.Printf("Measuring %d Warm Starts...\n", warmRuns)
	var totalWarm time.Duration
	var minWarm, maxWarm time.Duration
	var count429 int
	var successfulRuns int

	for i := 0; i < warmRuns; i++ {
		lat, status := measureInvoke()
		if status == 429 {
			count429++
			continue
		}
		if status != 200 {
			fmt.Printf("⚠️  Unexpected status: %d\n", status)
			continue
		}

		successfulRuns++
		totalWarm += lat
		if successfulRuns == 1 || lat < minWarm {
			minWarm = lat
		}
		if lat > maxWarm {
			maxWarm = lat
		}
	}

	fmt.Printf("\n📊 Benchmark Results (Warm Starts):\n")
	if successfulRuns > 0 {
		avgWarm := totalWarm / time.Duration(successfulRuns)
		fmt.Printf("   Average: %v\n", avgWarm)
		fmt.Printf("   Min:     %v\n", minWarm)
		fmt.Printf("   Max:     %v\n", maxWarm)
	}
	fmt.Printf("   Successful Runs: %d/%d\n", successfulRuns, warmRuns)
	if count429 > 0 {
		fmt.Printf("   Rate Limited (429): %d\n", count429)
	}
}

func deploy(rateLimit int) error {
	file, err := os.Open(zipPath)
	if err != nil {
		return err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filepath.Base(zipPath))
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, file); err != nil {
		return err
	}

	if err := writer.WriteField("rateLimit", strconv.Itoa(rateLimit)); err != nil {
		return err
	}

	writer.Close()

	req, err := http.NewRequest("POST", baseURL+"/deploy", body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

func measureInvoke() (time.Duration, int) {
	start := time.Now()

	payload := InvokeRequest{
		Body: `{"test": true}`,
	}
	jsonBody, _ := json.Marshal(payload)

	resp, err := http.Post(baseURL+"/invoke/"+funcName, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return 0, 0
	}
	defer resp.Body.Close()

	return time.Since(start), resp.StatusCode
}

func cleanup() {
	req, _ := http.NewRequest("DELETE", baseURL+"/del/"+funcName, nil)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err == nil {
		resp.Body.Close()
	}
}

func ensureZip() error {
	_, err := os.Stat(zipPath)
	return err
}
