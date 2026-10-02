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

	"runtime/debug"

	"github.com/eswar-7116/glambdar/v3/internal/api"
	"github.com/eswar-7116/glambdar/v3/internal/auth"
	"github.com/eswar-7116/glambdar/v3/internal/config"
	"github.com/eswar-7116/glambdar/v3/internal/functions"
	"github.com/spf13/cobra"
)

var VERSION = getVersion()

func getVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	if info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}

	for _, s := range info.Settings {
		if s.Key == "vcs.revision" {
			rev := s.Value
			if len(rev) > 7 {
				rev = rev[:7]
			}
			return "dev-" + rev
		}
	}
	return "dev"
}

const PORT = "8000"

var (
	flagDBType            string
	flagDSN               string
	flagS3Endpoint        string
	flagS3Region          string
	flagS3Bucket          string
	flagS3AccessKeyID     string
	flagS3SecretAccessKey string
	flagS3SessionToken    string
	flagS3ForcePathStyle  bool
)

var RootCmd = &cobra.Command{
	Use:     "glambdar [command]",
	Short:   "Glambdar serverless execution engine",
	Version: VERSION,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		applyPersistentFlags(cmd)
	},
	Run: func(cmd *cobra.Command, args []string) {
		Init()
		Start()
	},
}

func applyPersistentFlags(cmd *cobra.Command) {
	if cmd.Flags().Changed("db-type") {
		config.Overrides.DBType = flagDBType
	}
	if cmd.Flags().Changed("dsn") {
		config.Overrides.DSN = flagDSN
	}
	if cmd.Flags().Changed("s3-endpoint") {
		config.Overrides.S3Endpoint = flagS3Endpoint
	}
	if cmd.Flags().Changed("s3-region") {
		config.Overrides.S3Region = flagS3Region
	}
	if cmd.Flags().Changed("s3-bucket") {
		config.Overrides.S3Bucket = flagS3Bucket
	}
	if cmd.Flags().Changed("s3-access-key-id") {
		config.Overrides.S3AccessKeyID = flagS3AccessKeyID
	}
	if cmd.Flags().Changed("s3-secret-access-key") {
		config.Overrides.S3SecretAccessKey = flagS3SecretAccessKey
	}
	if cmd.Flags().Changed("s3-session-token") {
		config.Overrides.S3SessionToken = flagS3SessionToken
	}
	if cmd.Flags().Changed("s3-force-path-style") {
		config.Overrides.S3ForcePathStyle = &flagS3ForcePathStyle
	}
}

func init() {
	RootCmd.PersistentFlags().StringVar(&flagDBType, "db-type", "", "Database type (postgres, mysql)")
	RootCmd.PersistentFlags().StringVar(&flagDSN, "dsn", "", "Database DSN")
	RootCmd.PersistentFlags().StringVar(&flagS3Endpoint, "s3-endpoint", "", "S3 endpoint URL")
	RootCmd.PersistentFlags().StringVar(&flagS3Region, "s3-region", "", "S3 region")
	RootCmd.PersistentFlags().StringVar(&flagS3Bucket, "s3-bucket", "", "S3 bucket name")
	RootCmd.PersistentFlags().StringVar(&flagS3AccessKeyID, "s3-access-key-id", "", "S3 access key ID")
	RootCmd.PersistentFlags().StringVar(&flagS3SecretAccessKey, "s3-secret-access-key", "", "S3 secret access key")
	RootCmd.PersistentFlags().StringVar(&flagS3SessionToken, "s3-session-token", "", "S3 session token")
	RootCmd.PersistentFlags().BoolVar(&flagS3ForcePathStyle, "s3-force-path-style", false, "Force path style for S3")

	RootCmd.AddCommand(agentCmd)
	RootCmd.AddCommand(controllerCmd)
	RootCmd.AddCommand(auth.ResetAdminKeyCmd)
}

func Execute() error {
	return RootCmd.Execute()
}

func Init() {
	// Set the required file paths
	if err := config.InitPaths(); err != nil {
		fmt.Println(err.Error())
		fmt.Println("Failed to initialize paths.")
		os.Exit(1)
	}

	if err := config.DB.AutoMigrate(&functions.Metadata{}, &functions.Log{}, &auth.APIKey{}, &auth.AuditLog{}); err != nil {
		fmt.Println("Failed to migrate database schema:", err)
		os.Exit(1)
	}

	if err := auth.BootstrapRootKey(); err != nil {
		fmt.Println("Failed to bootstrap admin key:", err)
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
	functions.FlushInvokeCounters()

	// Clean up all containers
	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cleanupCancel()

	config.PoolManager.DeleteAllContainers(cleanupCtx, config.DockerClient)

	if err := config.DockerClient.Close(); err != nil {
		fmt.Printf("Error closing Docker client: %v\n", err)
	}
}
