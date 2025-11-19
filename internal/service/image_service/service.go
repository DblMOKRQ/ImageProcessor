package image_service

import (
	"ImageProcessor/internal/models"
	"context"
	"fmt"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"io"
	"path/filepath"
	"time"
)

type Repo interface {
	CreateTask(ctx context.Context, task *models.Task) error
	UpdateStatus(ctx context.Context, id string, status models.TaskStatus) error
	GetTask(ctx context.Context, id string) (*models.Task, error)
	DeleteTask(ctx context.Context, id string) error
}

type FileStorage interface {
	Save(path string, image io.Reader) error
	Delete(path string) error
}

type Produce interface {
	Publish(ctx context.Context, task *models.ProcessingCommand) error
}

type ImageService struct {
	repo    Repo
	storage FileStorage
	produce Produce
	log     *zap.Logger
}

func NewImageService(repo Repo, storage FileStorage, produce Produce, log *zap.Logger) *ImageService {
	return &ImageService{
		repo:    repo,
		storage: storage,
		produce: produce,
		log:     log.Named("service"),
	}
}

func (s *ImageService) UploadImage(ctx context.Context, image io.Reader, extension string) (string, error) {
	id := uuid.New().String()
	imagePath := fmt.Sprintf(models.OriginalPath, id, extension)

	err := s.storage.Save(imagePath, image)
	if err != nil {
		s.log.Error("failed to save image", zap.String("imagePath", imagePath), zap.Error(err))
		return "", fmt.Errorf("failed to save image: %w", err)
	}

	task := &models.Task{
		ID:                  id,
		Status:              models.StatusProcessing,
		OriginalPath:        imagePath,
		RequestedOperations: models.RequestedOperations,
		CreatedAt:           time.Now(),
	}

	err = s.repo.CreateTask(ctx, task)
	if err != nil {
		s.log.Error("failed to create task, compensating by deleting file", zap.String("imagePath", imagePath), zap.Error(err))
		if cleanupErr := s.storage.Delete(imagePath); cleanupErr != nil {
			s.log.Error("failed to cleanup (delete) file after db error", zap.String("imagePath", imagePath), zap.NamedError("cleanup_error", cleanupErr))
		}
		return "", fmt.Errorf("failed to create task: %w", err)
	}
	processingMessage := &models.ProcessingCommand{
		ID:                  task.ID,
		OriginalPath:        task.OriginalPath,
		RequestedOperations: task.RequestedOperations,
		CreatedAt:           task.CreatedAt,
	}
	err = s.produce.Publish(ctx, processingMessage)
	if err != nil {
		s.log.Error("failed to publish task, compensating by deleting file and db record", zap.String("imagePath", imagePath), zap.Error(err))

		if cleanupErr := s.storage.Delete(imagePath); cleanupErr != nil {
			s.log.Error("failed to cleanup (delete) file after publish error", zap.String("imagePath", imagePath), zap.NamedError("cleanup_error", cleanupErr))
		}
		if cleanupErr := s.repo.DeleteTask(ctx, id); cleanupErr != nil {
			s.log.Error("failed to cleanup (delete) db task after publish error", zap.String("id", id), zap.NamedError("cleanup_error", cleanupErr))
		}
		return "", fmt.Errorf("failed to publish task: %w", err)
	}
	return id, nil
}

func (s *ImageService) GetImage(ctx context.Context, id string) (*models.Task, error) {
	task, err := s.repo.GetTask(ctx, id)
	if err != nil {
		s.log.Error("failed to get task", zap.String("id", id), zap.Error(err))
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	return task, nil
}

func (s *ImageService) DeleteImage(ctx context.Context, id string) error {

	task, err := s.repo.GetTask(ctx, id)
	if err != nil {
		s.log.Warn("task not found on delete, possibly already deleted", zap.String("id", id), zap.Error(err))
		return nil
	}

	s.log.Info("Deleting original file", zap.String("path", task.OriginalPath))
	if err := s.storage.Delete(task.OriginalPath); err != nil {
		s.log.Error("failed to delete original file, continuing cleanup", zap.String("path", task.OriginalPath), zap.Error(err))
	}

	baseFilename := filepath.Base(task.OriginalPath)
	for _, operation := range models.RequestedOperations {
		processedPath := fmt.Sprintf(models.ProcessPath, operation, baseFilename)
		s.log.Info("Deleting processed file", zap.String("path", processedPath))
		if err := s.storage.Delete(processedPath); err != nil {
			s.log.Error("failed to delete processed file", zap.String("path", processedPath), zap.Error(err))
		}
	}

	err = s.repo.DeleteTask(ctx, id)
	if err != nil {
		s.log.Error("failed to delete task from db after cleaning files", zap.String("id", id), zap.Error(err))
		return fmt.Errorf("failed to delete task record: %w", err)
	}

	s.log.Info("Successfully deleted task and associated files", zap.String("id", id))
	return nil
}
