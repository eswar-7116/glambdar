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

	"github.com/eswar-7116/glambdar/v3/internal/api"
	"github.com/eswar-7116/glambdar/v3/internal/auth"
	"github.com/eswar-7116/glambdar/v3/internal/cluster"
	"github.com/eswar-7116/glambdar/v3/internal/config"
	"github.com/eswar-7116/glambdar/v3/internal/controller"
	"github.com/eswar-7116/glambdar/v3/internal/functions"
	"github.com/eswar-7116/glambdar/v3/internal/ratelimit"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/cobra"
)

var (
	controllerHTTPPort  string
	controllerRedisAddr string
)

var controllerCmd = &cobra.Command{
	Use:   "controller",
	Short: "Run controller (control plane HTTP server)",
	RunE: func(cmd *cobra.Command, args []string) error {
		runController(cmd)
		return nil
	},
}

func init() {
	controllerCmd.Flags().StringVar(&controllerHTTPPort, "http-port", "8000", "HTTP server port")
	controllerCmd.Flags().StringVar(&controllerRedisAddr, "redis-addr", "", "Redis address for cluster coordination (e.g. localhost:6379)")
}

func runController(cmd *cobra.Command) {
	if err := config.InitControllerPaths(); err != nil {
		fmt.Println(err.Error())
		fmt.Println("Failed to initialize controller paths.")
		os.Exit(1)
	}

	cfg, _ := config.LoadConfig()
	if cfg == nil {
		cfg = &config.Config{}
	}

	// Apply CLI flag overrides
	if cmd.Flags().Changed("http-port") {
		cfg.HTTPPort = controllerHTTPPort
	} else if cfg.HTTPPort != "" {
		controllerHTTPPort = cfg.HTTPPort
	}

	if controllerRedisAddr != "" {
		cfg.RedisAddr = controllerRedisAddr
	}

	// redis_addr is required for controller mode
	if cfg.RedisAddr == "" {
		fmt.Println("Error: redis_addr is required for controller mode.")
		fmt.Println("Set it via --redis-addr flag, config.json, or GLMBD_REDIS_ADDR environment variable.")
		os.Exit(1)
	}

	// Database migrations
	if err := config.DB.AutoMigrate(&functions.Metadata{}, &functions.Log{}, &auth.APIKey{}, &auth.AuditLog{}); err != nil {
		fmt.Println("Failed to migrate database schema:", err)
		os.Exit(1)
	}

	if err := auth.BootstrapRootKey(); err != nil {
		fmt.Println("Failed to bootstrap admin key:", err)
		os.Exit(1)
	}

	// Connect to Redis cluster state
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stateProvider := cluster.NewRedisStateProvider(cfg.RedisAddr)
	if err := stateProvider.Join(ctx); err != nil {
		fmt.Printf("Failed to connect to Redis at %s: %v\n", cfg.RedisAddr, err)
		os.Exit(1)
	}
	log.Printf("Connected to Redis cluster state at %s", cfg.RedisAddr)

	// Initialize global rate limiter backed by Redis
	redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	config.RateLimiter = ratelimit.NewRedisLimiter(redisClient)

	// Initialize cluster router and gRPC client pool
	router := cluster.NewRouter(stateProvider)
	grpcClientPool := controller.NewGRPCClientPool()

	// Inject controller mode into API handlers
	api.SetControllerMode(router, grpcClientPool, stateProvider)

	// Start HTTP server
	srv := &http.Server{
		Addr:    ":" + controllerHTTPPort,
		Handler: api.Router(),
	}

	log.Printf("Glambdar Controller listening on HTTP port %s...", controllerHTTPPort)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Println("Error while starting the server:", err)
		}
	}()

	// Signal handling
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	<-stop

	// Graceful shutdown
	fmt.Println("\nShutting down controller...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		fmt.Println("Server shutdown failed:", err)
	}

	functions.FlushInvokeCounters()
	grpcClientPool.CloseAll()
	stateProvider.Leave(ctx)
}
