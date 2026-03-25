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
)

const PORT = "8000"

func Init() {
	// Set the required file paths
	if err := config.InitPaths(); err != nil {
		fmt.Println(err.Error())
		fmt.Println("Please make sure you defined GLAMBDAR_DIR in the environment.")
		os.Exit(1)
	}
}

func Start() {
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

	// Signal handling
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	<-stop

	fmt.Println("\nShutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		fmt.Println("Server Shutdown Failed:", err)
	}

	if err := config.DockerClient.Close(); err != nil {
		fmt.Printf("Error closing Docker client: %v\n", err)
	}

	os.Remove(config.UDSPath)
	fmt.Println("Deleting the UDS...")
}
