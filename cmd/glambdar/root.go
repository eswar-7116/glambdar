package glambdar

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eswar-7116/glambdar/internal/api"
	"github.com/eswar-7116/glambdar/internal/config"
	"github.com/eswar-7116/glambdar/internal/functions"
)

const VERSION = "v3.0.0"

const PORT = "8000"

func Init() {
	// Set the required file paths
	if err := config.InitPaths(); err != nil {
		fmt.Println(err.Error())
		fmt.Println("Please make sure you defined GLAMBDAR_DIR in the environment.")
		os.Exit(1)
	}

	if err := config.DB.AutoMigrate(&functions.Metadata{}, &functions.Log{}); err != nil {
		fmt.Println("Failed to migrate database schema:", err)
		os.Exit(1)
	}
}

func Start() {
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()
	if err := config.DockerClient.Ping(pingCtx); err != nil {
		fmt.Println("Error: Docker daemon is not reachable. Please make sure Docker is running.")
		fmt.Printf("Details: %v\n", err)
		os.Exit(1)
	}

	log.Println("Glambdar is running on port 8000")
	srv := &http.Server{
		Addr:    ":" + PORT,
		Handler: api.Router(),
	}

	// Start server in a goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Println("Error while starting the server: ", err)
		}
	}()

	// Context for eviction goroutine
	evictCtx, evictCancel := context.WithCancel(context.Background())
	defer evictCancel()

	// Start predictive prewarmer
	config.PoolManager.StartPrewarmer(evictCtx, config.DockerClient, config.FunctionsDir, 30*time.Second)

	// Start CRON job to clean stale containers
	cronTicker := time.NewTicker(30 * time.Second)
	defer cronTicker.Stop()

	go func() {
		for range cronTicker.C {
			config.PoolManager.RemoveStaleContainers(evictCtx, config.DockerClient, 10*time.Minute)
		}
	}()

	// Signal handling
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	<-stop

	// Graceful HTTP shutdown with timeout
	fmt.Println("\nShutting down server...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		fmt.Println("Server shutdown failed:", err)
	}

	// Clean up all containers
	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cleanupCancel()

	config.PoolManager.DeleteAllContainers(cleanupCtx, config.DockerClient)

	if err := config.DockerClient.Close(); err != nil {
		fmt.Printf("Error closing Docker client: %v\n", err)
	}
}
