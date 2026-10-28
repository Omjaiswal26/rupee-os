package main

import (
	"log"
	"os"

	"github.com/hibiken/asynq"
	"github.com/rupee-os/rupee-os/internal/classification"
	"github.com/rupee-os/rupee-os/internal/config"
	"github.com/rupee-os/rupee-os/internal/database"
	"github.com/rupee-os/rupee-os/internal/logging"
	"github.com/rupee-os/rupee-os/internal/services"
	"github.com/rupee-os/rupee-os/internal/worker"
)

func main() {
	cfg := config.Load()
	logger := logging.New(cfg.LogLevel)

	if err := os.MkdirAll(cfg.UploadDir, 0o750); err != nil {
		log.Fatalf("upload dir: %v", err)
	}

	db, err := database.Connect(cfg.DatabaseURL, logger)
	if err != nil {
		log.Fatalf("db: %v", err)
	}

	redisOpt := asynq.RedisClientOpt{Addr: cfg.RedisAddr, Password: cfg.RedisPassword}
	asynqClient := asynq.NewClient(redisOpt)
	defer asynqClient.Close()

	importSvc := &services.ImportService{
		DB:        db,
		Asynq:     asynqClient,
		UploadDir: cfg.UploadDir,
		Log:       logger,
	}
	classifySvc := &services.ClassifyService{
		DB:          db,
		Client:      classification.NewClient(cfg.OllamaBaseURL, cfg.OllamaModel, cfg.ClassificationMinConfidence),
		Log:         logger,
		Concurrency: cfg.ClassifyConcurrency,
	}

	srv := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency: 2,
		Queues: map[string]int{
			"default": 10,
		},
	})

	mux := asynq.NewServeMux()
	h := &worker.Handlers{
		Imports:  importSvc,
		Classify: classifySvc,
		Log:      logger,
	}
	h.Register(mux)

	logger.Info("worker starting", "redis", cfg.RedisAddr, "model", cfg.OllamaModel, "classify_concurrency", cfg.ClassifyConcurrency)
	if err := srv.Run(mux); err != nil {
		log.Fatal(err)
	}
}
