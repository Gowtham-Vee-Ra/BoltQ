package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"BoltQ/internal/api"
	"BoltQ/internal/job"
	"BoltQ/internal/queue"
	"BoltQ/pkg/config"
	"BoltQ/pkg/logger"
	"BoltQ/pkg/metrics"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/cors"
)

func main() {
	log := logger.NewLogger("api")
	log.Info("Starting BoltQ API Service...")

	if err := godotenv.Load(); err != nil {
		log.Error("No .env file found or couldn't load it")
	}

	apiPort := config.GetEnv("API_PORT", "8080")
	metricsPort := config.GetEnv("METRICS_PORT", "9093")
	redisAddr := config.GetEnv("REDIS_ADDR", "localhost:6379")
	allowedOrigin := config.GetEnv("ALLOWED_ORIGIN", "http://localhost:5173")
	apiKey := config.GetEnv("API_KEY", "")
	rateLimitRPS := config.GetEnvAsInt("RATE_LIMIT_RPS", 20)
	rateLimitBurst := config.GetEnvAsInt("RATE_LIMIT_BURST", 40)

	redisClient := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Error(fmt.Sprintf("Failed to connect to Redis: %v", err))
		os.Exit(1)
	}
	log.Info(fmt.Sprintf("Connected to Redis at %s", redisAddr))

	metricsCollector := metrics.NewMetricsCollector("api")
	redisQueue := queue.NewRedisQueue(redisClient, log)
	workflowManager := job.NewWorkflowManager(redisClient, log)

	websocketManager := api.NewWebSocketManager(redisClient, log, allowedOrigin)
	websocketManager.Start()

	apiHandler := api.NewHandler(redisQueue, log, metricsCollector, workflowManager)

	router := mux.NewRouter()

	// Rate limit every request by client IP, then require an API key on
	// mutating requests. Order matters: limit before auth so unauthenticated
	// floods are shed cheaply.
	router.Use(api.NewRateLimiter(float64(rateLimitRPS), float64(rateLimitBurst)).Middleware())
	router.Use(api.APIKeyAuth(apiKey, log))

	if apiKey == "" {
		log.Error("API_KEY is not set — mutating endpoints (job submission, workflows) are UNAUTHENTICATED. Set API_KEY before exposing this service.")
	} else {
		log.Info("API key authentication enabled for mutating endpoints")
	}

	apiHandler.RegisterRoutes(router)
	router.HandleFunc("/ws/jobs", websocketManager.HandleJobUpdatesWebSocket)

	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{allowedOrigin},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}).Handler(router)

	apiServer := &http.Server{
		Addr:         ":" + apiPort,
		Handler:      corsHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	metricsRouter := mux.NewRouter()
	metricsRouter.Handle("/metrics", promhttp.Handler())

	metricsServer := &http.Server{
		Addr:    ":" + metricsPort,
		Handler: metricsRouter,
	}

	go func() {
		log.Info(fmt.Sprintf("API server listening on port %s", apiPort))
		if err := apiServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error(fmt.Sprintf("Error starting API server: %v", err))
			os.Exit(1)
		}
	}()

	go func() {
		log.Info(fmt.Sprintf("Metrics server listening on port %s", metricsPort))
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error(fmt.Sprintf("Error starting metrics server: %v", err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Shutting down servers...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := apiServer.Shutdown(shutdownCtx); err != nil {
		log.Error(fmt.Sprintf("API server shutdown error: %v", err))
	}
	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		log.Error(fmt.Sprintf("Metrics server shutdown error: %v", err))
	}

	websocketManager.Stop()
	log.Info("Servers stopped")
}
