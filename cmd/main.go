package main

import (
	storage "ImageProcessor/internal/file_storage"
	"ImageProcessor/internal/messagebroker"
	"ImageProcessor/internal/models"
	"ImageProcessor/internal/modifer"
	"ImageProcessor/internal/repository"
	"ImageProcessor/internal/service/image_service"
	"ImageProcessor/internal/service/worker_service"
	"ImageProcessor/internal/transport/handlers"
	"ImageProcessor/internal/transport/router"
	"ImageProcessor/pkg/logger"
	"context"
	"errors"
	"github.com/wb-go/wbf/config"
	"go.uber.org/zap"
	"net/http"
)

func main() {

	ctx := context.Background()
	cfg := config.New()
	_ = cfg.LoadConfigFiles("./config/config.yaml")
	log, err := logger.NewLogger(cfg.GetString("log_level"))
	if err != nil {
		panic(err)
	}
	defer log.Sync()
	repo, err := repository.NewRepository(cfg.GetString("master_dsn"), cfg.GetStringSlice("slaveDSNs"), log)
	if err != nil {
		log.Fatal("failed to init repository", zap.Error(err))
	}
	fileStorage, err := storage.NewFileStorage(models.BasePath, log)
	if err != nil {
		log.Fatal("failed to init file storage", zap.Error(err))
	}

	produce := messagebroker.NewProducer(cfg.GetStringSlice("brokers"), cfg.GetString("topic"), log)

	service := image_service.NewImageService(repo, fileStorage, produce, log)

	modif, err := modifer.NewModifier(cfg.GetString("watermarkPath"), models.BasePath, fileStorage, log)

	if err != nil {
		log.Fatal("failed to init modifier", zap.Error(err))
	}

	consum := messagebroker.NewConsumer(cfg.GetStringSlice("brokers"), cfg.GetString("topic"), cfg.GetString("group_id"), log)

	defer consum.Close()

	worker := worker_service.NewWorker(consum, modif, repo, log)

	go func() {
		if err := worker.Starting(ctx); err != nil {
			log.Fatal("failed to start worker", zap.Error(err))
		}
	}()

	imageHandlers := handlers.NewImageHandler(service)

	rout := router.NewRouter(cfg.GetString("log_level"), imageHandlers, log)
	srv := &http.Server{
		Addr:    cfg.GetString("addr"),
		Handler: rout.GetEngine(),
	}
	log.Info("Starting server", zap.String("addr", srv.Addr))
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal("Failed to listen and server", zap.Error(err))
	}
}
