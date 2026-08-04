package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"taskservice/internal/auth"
	"taskservice/internal/config"
	"taskservice/internal/db"
	"taskservice/internal/handlers"
	"taskservice/internal/middleware"
	"taskservice/internal/repository"
	"taskservice/internal/service"
)

func main() {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	mysqlConn, err := db.NewMySQL(cfg.MySQL.DSN, cfg.MySQL.MaxOpenConn, cfg.MySQL.MaxIdleConn)
	if err != nil {
		log.Fatalf("failed to connect to mysql: %v", err)
	}
	defer mysqlConn.Close()

	redisClient, err := db.NewRedis(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}
	defer redisClient.Close()

	middleware.RegisterMetrics()

	jwtManager := auth.NewJWTManager(cfg.JWT.Secret, cfg.JWT.TTLMin)

	userRepo := repository.NewUserRepository(mysqlConn)
	teamRepo := repository.NewTeamRepository(mysqlConn)
	taskRepo := repository.NewTaskRepository(mysqlConn, redisClient)

	emailService := service.NewEmailService()
	userService := service.NewUserService(userRepo, jwtManager)
	teamService := service.NewTeamService(teamRepo, userRepo, emailService)
	taskService := service.NewTaskService(taskRepo, teamRepo)

	rateLimiter := middleware.NewRateLimiter(cfg.RateLimit.RequestsPerMinute)
	rateLimiter.Cleanup(time.Hour)

	router := &handlers.Router{
		JWT:       jwtManager,
		Auth:      handlers.NewAuthHandler(userService),
		Teams:     handlers.NewTeamHandler(teamService),
		Tasks:     handlers.NewTaskHandler(taskService),
		RateLimit: rateLimiter,
	}

	srv := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: router.Build(),
	}

	go func() {
		log.Printf("server listening on :%s", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("graceful shutdown failed: %v", err)
	}

	log.Println("server stopped")
}
