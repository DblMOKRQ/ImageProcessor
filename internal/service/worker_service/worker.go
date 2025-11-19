package worker_service

import (
	"ImageProcessor/internal/models"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	kafkaGo "github.com/segmentio/kafka-go"
	"go.uber.org/zap"
	"path/filepath"
)

type Consume interface {
	FetchMessage(ctx context.Context) (kafkaGo.Message, error)
	CommitMessage(ctx context.Context, msg kafkaGo.Message) error
	Close() error
}

type Modifier interface {
	Resize(sourcePath, targetPath string, width, height uint) error
	Thumbnail(sourcePath, targetPath string, maxWidth, maxHeight uint) error
	Watermark(sourcePath, targetPath string) error
}

type Repo interface {
	UpdateStatus(ctx context.Context, id string, status models.TaskStatus) error
}

type Worker struct {
	consume  Consume
	modifier Modifier
	repo     Repo
	log      *zap.Logger
}

func NewWorker(consume Consume, modifier Modifier, repo Repo, log *zap.Logger) *Worker {
	return &Worker{
		consume:  consume,
		modifier: modifier,
		repo:     repo,
		log:      log.Named("worker"),
	}
}

func (w *Worker) Starting(ctx context.Context) error {
	w.log.Info("Starting worker loop")
	for {
		select {
		case <-ctx.Done():
			w.log.Info("Worker shutting down due to context cancellation")
			return ctx.Err()
		default:
			msg, err := w.consume.FetchMessage(ctx)
			if err != nil {
				if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
					if err.Error() != "EOF" { // kafka-go возвращает EOF, если нет новых сообщений
						w.log.Warn("Failed to fetch message", zap.Error(err))
					}
				}
				continue
			}

			task, err := w.GetTask(msg)
			if err != nil {
				w.log.Warn("Error reading message", zap.Error(err))
				_ = w.consume.CommitMessage(ctx, msg)
				continue
			}
			for _, operation := range task.RequestedOperations {
				baseFilename := filepath.Base(task.OriginalPath)
				processedPath := fmt.Sprintf(models.ProcessPath, operation, baseFilename)
				switch operation {
				case "resize":
					if err := w.modifier.Resize(task.OriginalPath, processedPath, models.ResizeToWidth, models.ResizeToHeight); err != nil {
						w.failed(ctx, task.ID)
						w.log.Error("Error resizing image", zap.Error(err))
					}
				case "thumbnail":
					if err := w.modifier.Thumbnail(task.OriginalPath, processedPath, models.ThumbnailToWidth, models.ThumbnailToHeight); err != nil {
						w.failed(ctx, task.ID)
						w.log.Error("Error thumbnailing image", zap.Error(err))
					}
				case "watermark":
					if err := w.modifier.Watermark(task.OriginalPath, processedPath); err != nil {
						w.failed(ctx, task.ID)
						w.log.Error("Error watermarking image", zap.Error(err))
					}
				}
			}

			_ = w.consume.CommitMessage(ctx, msg)
			if err := w.repo.UpdateStatus(ctx, task.ID, models.StatusComplete); err != nil {
				w.log.Error("Error updating status", zap.Error(err))
			}
		}
	}
}

func (w *Worker) GetTask(msg kafkaGo.Message) (*models.Task, error) {
	w.log.Debug("Getting task", zap.ByteString("msg", msg.Value))
	var task models.Task
	err := json.Unmarshal(msg.Value, &task)
	if err != nil {
		w.log.Error("Failed to unmarshal task", zap.String("task", string(msg.Value)), zap.Error(err))
		return nil, fmt.Errorf("failed to unmarshal task: %w", err)
	}
	w.log.Debug("Got task", zap.String("task", string(msg.Value)))
	return &task, nil
}

func (w *Worker) failed(ctx context.Context, id string) {
	w.repo.UpdateStatus(ctx, id, models.StatusFailed)
}
