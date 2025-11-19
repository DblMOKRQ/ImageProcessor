package storage

import (
	"fmt"
	"go.uber.org/zap"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
)

type FileStorage struct {
	basePath string
	log      *zap.Logger
}

func NewFileStorage(basePath string, log *zap.Logger) (*FileStorage, error) {
	info, err := os.Stat(basePath)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(basePath, 0755); err != nil {
			return nil, fmt.Errorf("cannot create base directory %s: %w", basePath, err)
		}
	} else if !info.IsDir() {
		return nil, fmt.Errorf("base path %s is not a directory", basePath)
	}

	return &FileStorage{
		basePath: basePath,
		log:      log.Named("file"),
	}, nil
}

func (fs *FileStorage) Save(path string, image io.Reader) error {
	fullPath := filepath.Join(fs.basePath, path)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		fs.log.Error("Failed to create directory", zap.String("dir", dir), zap.Error(err))
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	file, err := os.Create(fullPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", fullPath, err)
	}
	defer file.Close()

	_, err = io.Copy(file, image)
	if err != nil {
		return fmt.Errorf("failed to write data to file %s: %w", fullPath, err)
	}

	return nil
}

func (fs *FileStorage) Delete(path string) error {
	fullPath := filepath.Join(fs.basePath, path)
	fs.log.Info("Attempting to delete file", zap.String("path", fullPath))

	err := os.Remove(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			fs.log.Warn("Attempted to delete a file that does not exist", zap.String("path", fullPath))
			return nil
		}
		fs.log.Error("Failed to delete file", zap.String("path", fullPath), zap.Error(err))
		return fmt.Errorf("failed to delete file %s: %w", fullPath, err)
	}

	fs.log.Info("Successfully deleted file", zap.String("path", fullPath))
	return nil
}

// LoadImage загружает файл с диска и декодирует его в image.Image.
func (fs *FileStorage) LoadImage(path string) (image.Image, string, error) {
	fullPath := filepath.Join(fs.basePath, path)
	file, err := os.Open(fullPath)
	if err != nil {
		fs.log.Error("Failed to open image file", zap.String("path", fullPath), zap.Error(err))
		return nil, "", fmt.Errorf("failed to open file %s: %w", fullPath, err)
	}
	defer file.Close()

	// image.Decode сам определяет формат (jpeg, png, gif) по сигнатуре файла
	img, format, err := image.Decode(file)
	if err != nil {
		fs.log.Error("Failed to decode image", zap.String("path", fullPath), zap.Error(err))
		return nil, "", fmt.Errorf("failed to decode image %s: %w", fullPath, err)
	}

	fs.log.Debug("Successfully loaded image", zap.String("path", fullPath), zap.String("format", format))
	return img, format, nil
}

// SaveImage сохраняет image.Image в файл, кодируя его в нужный формат.
func (fs *FileStorage) SaveImage(path string, img image.Image, format string) error {
	fullPath := filepath.Join(fs.basePath, path)

	// Убедимся, что директория для сохранения существует
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		fs.log.Error("Failed to create directory for saving image", zap.String("path", fullPath), zap.Error(err))
		return fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := os.Create(fullPath)
	if err != nil {
		fs.log.Error("Failed to create file for saving image", zap.String("path", fullPath), zap.Error(err))
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Кодируем изображение в зависимости от его оригинального формата
	switch format {
	case "jpeg":
		return jpeg.Encode(file, img, &jpeg.Options{Quality: 90})
	case "png":
		return png.Encode(file, img)
	case "gif":
		return gif.Encode(file, img, nil)
	default:
		fs.log.Error("Unsupported image format for saving", zap.String("format", format))
		return fmt.Errorf("unsupported format for saving: %s", format)
	}
}
