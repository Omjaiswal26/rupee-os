package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"github.com/rupee-os/rupee-os/internal/classification"
	"github.com/rupee-os/rupee-os/internal/config"
	"github.com/rupee-os/rupee-os/internal/database"
	"github.com/rupee-os/rupee-os/internal/handlers"
	"github.com/rupee-os/rupee-os/internal/logging"
	"github.com/rupee-os/rupee-os/internal/services"
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
	if err := database.Migrate(cfg.DatabaseURL, cfg.MigrationsPath, logger); err != nil {
		log.Fatalf("migrate: %v", err)
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
	txnSvc := &services.TransactionService{DB: db}
	healthSvc := &services.HealthService{
		DB:     db,
		Txns:   txnSvc,
		Client: classification.NewHealthClient(cfg.OllamaBaseURL, cfg.OllamaHealthModel),
		Log:    logger,
	}

	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(handlers.CORS(cfg.CORSOrigin))
	r.MaxMultipartMemory = int64(cfg.MaxUploadMB) << 20

	api := &handlers.API{
		Imports:     importSvc,
		Txns:        txnSvc,
		MonthHealth: healthSvc,
		Cfg:         cfg,
		Log:         logger,
	}
	api.Register(r)

	logger.Info("api listening", "addr", cfg.APIAddr)
	if err := r.Run(cfg.APIAddr); err != nil {
		log.Fatal(err)
	}
}
