package modifer

import (
	"fmt"
	"github.com/nfnt/resize"
	"go.uber.org/zap"
	"image"
	"image/draw"
	"os"
)

type Storage interface {
	SaveImage(path string, img image.Image, format string) error
	LoadImage(path string) (image.Image, string, error)
}

type Modifier struct {
	watermarkImage image.Image
	basePath       string
	storage        Storage
	log            *zap.Logger
}

// NewModifier создает новый экземпляр Modifier.
// watermarkPath - это путь к файлу, который будет использоваться как водяной знак (например, "assets/watermark.png").
func NewModifier(watermarkPath string, basePath string, storage Storage, logger *zap.Logger) (*Modifier, error) {
	wmFile, err := os.Open(watermarkPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open watermark file %s: %w", watermarkPath, err)
	}
	defer wmFile.Close()

	wmImg, _, err := image.Decode(wmFile)
	if err != nil {
		return nil, fmt.Errorf("failed to decode watermark image: %w", err)
	}

	logger.Info("Watermark image loaded successfully")

	return &Modifier{
		watermarkImage: wmImg,
		basePath:       basePath,
		storage:        storage,
		log:            logger.Named("modifier"),
	}, nil
}

// Resize изменяет размер изображения и сохраняет результат.
func (m *Modifier) Resize(sourcePath, targetPath string, width, height uint) error {
	img, format, err := m.storage.LoadImage(sourcePath)
	if err != nil {
		return err
	}

	resizedImg := resize.Resize(width, height, img, resize.Lanczos3)

	m.log.Info("Resized image", zap.String("target", targetPath))
	return m.storage.SaveImage(targetPath, resizedImg, format)
}

// Thumbnail создает миниатюру, сохраняя пропорции, и сохраняет результат.
func (m *Modifier) Thumbnail(sourcePath, targetPath string, maxWidth, maxHeight uint) error {
	img, format, err := m.storage.LoadImage(sourcePath)
	if err != nil {
		return err
	}

	thumbImg := resize.Thumbnail(maxWidth, maxHeight, img, resize.Lanczos3)

	m.log.Info("Created thumbnail", zap.String("target", targetPath))
	return m.storage.SaveImage(targetPath, thumbImg, format)
}

func (m *Modifier) Watermark(sourcePath, targetPath string) error {
	img, format, err := m.storage.LoadImage(sourcePath)
	if err != nil {
		return err
	}

	bounds := img.Bounds()

	newImg := image.NewRGBA(bounds)
	draw.Draw(newImg, bounds, img, image.Point{}, draw.Src)

	offset := image.Pt(bounds.Dx()-m.watermarkImage.Bounds().Dx()-10, bounds.Dy()-m.watermarkImage.Bounds().Dy()-10)

	draw.Draw(newImg, m.watermarkImage.Bounds().Add(offset), m.watermarkImage, image.Point{}, draw.Over)

	m.log.Info("Applied watermark", zap.String("target", targetPath))
	return m.storage.SaveImage(targetPath, newImg, format)
}
