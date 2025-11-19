package repository

import (
	"ImageProcessor/internal/models"
	"context"
	"errors"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/wb-go/wbf/dbpg"
	"go.uber.org/zap"
	"os"
	"path/filepath"
)

type Repository struct {
	db  *dbpg.DB
	log *zap.Logger
}

const (
	createQuery       = `INSERT INTO images (id,status,original_path,created_at) VALUES ($1,$2,$3,$4)`
	updateStatusQuery = `UPDATE images SET status = $1 WHERE id = $2`
	deleteQuery       = `DELETE FROM images WHERE id = $1`
	getQuery          = `SELECT id,status,original_path,created_at FROM images WHERE id = $1`
)

func NewRepository(masterDSN string, slaveDSNs []string, log *zap.Logger) (*Repository, error) {
	opts := dbpg.Options{
		MaxOpenConns: 10,
		MaxIdleConns: 5,
	}
	db, err := dbpg.New(masterDSN, slaveDSNs, &opts)
	if err != nil {
		log.Fatal("Failed to connect to database", zap.Error(err))
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Info("Starting database migrations")

	if err := runMigrations(masterDSN); err != nil {
		log.Error("Failed to run migrations", zap.Error(err))
		return nil, fmt.Errorf("failed to run migration: %w", err)
	}
	log.Info("Successfully migrated database")

	return &Repository{db: db, log: log.Named("repository")}, nil
}

func (r *Repository) CreateTask(ctx context.Context, task *models.Task) error {
	_, err := r.db.ExecWithRetry(ctx, models.RetryStrategy, createQuery, task.ID, task.Status, task.OriginalPath, task.CreatedAt)
	if err != nil {
		r.log.Error("Failed to create task", zap.Error(err))
		return fmt.Errorf("failed to create task: %w", err)
	}
	return nil
}

func (r *Repository) UpdateStatus(ctx context.Context, id string, status models.TaskStatus) error {
	r.log.Debug("Updating status", zap.String("id", id), zap.Any("status", status))
	_, err := r.db.ExecWithRetry(ctx, models.RetryStrategy, updateStatusQuery, status, id)
	if err != nil {
		r.log.Error("Failed to update status", zap.Error(err))
		return fmt.Errorf("failed to update status: %w", err)
	}
	r.log.Debug("Successfully updated status", zap.String("id", id), zap.Any("status", status))
	return nil
}
func (r *Repository) GetTask(ctx context.Context, id string) (*models.Task, error) {
	var task models.Task
	row, err := r.db.QueryRowWithRetry(ctx, models.RetryStrategy, getQuery, id)
	if err != nil {
		r.log.Error("Failed to get task", zap.Error(err))
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	err = row.Scan(&task.ID, &task.Status, &task.OriginalPath, &task.CreatedAt)
	if err != nil {
		r.log.Error("Failed to get task", zap.Error(err))
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	return &task, nil
}

func (r *Repository) DeleteTask(ctx context.Context, id string) error {
	_, err := r.db.ExecWithRetry(ctx, models.RetryStrategy, deleteQuery, id)
	if err != nil {
		r.log.Error("Failed to delete task", zap.Error(err))
		return fmt.Errorf("failed to delete task: %w", err)
	}
	return nil
}

func runMigrations(connStr string) error {
	migratePath := os.Getenv("MIGRATE_PATH")
	if migratePath == "" {
		migratePath = "./migrations"
	}
	absPath, err := filepath.Abs(migratePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}
	absPath = filepath.ToSlash(absPath)
	migrateUrl := fmt.Sprintf("file://%s", absPath)
	m, err := migrate.New(migrateUrl, connStr)
	if err != nil {
		return fmt.Errorf("start migrations error %v", err)
	}
	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			return nil
		}
		return fmt.Errorf("migration up error: %v", err)
	}
	return nil
}
