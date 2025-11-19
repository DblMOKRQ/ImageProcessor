package handlers

import (
	"ImageProcessor/internal/models"
	"ImageProcessor/internal/service/image_service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"
	"path/filepath"
)

type ImageHandler struct {
	imageService *image_service.ImageService
}

func NewImageHandler(imageService *image_service.ImageService) *ImageHandler {
	return &ImageHandler{imageService: imageService}
}

func (h *ImageHandler) UploadImage(c *gin.Context) {
	log := c.MustGet("logger").(*zap.Logger)
	log.Debug("Uploading Image")
	fileHeader, err := c.FormFile("image")
	if err != nil {
		log.Error("Failed to get file", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get file"})
		return
	}
	extension := filepath.Ext(fileHeader.Filename)
	if extension == "" {
		log.Error("Failed to get file extension", zap.String("filename", fileHeader.Filename))
		c.JSON(http.StatusBadRequest, gin.H{"error": "file must have an extension"})
		return
	}

	if !models.AllowedExtensions[extension] {
		log.Warn("File extension is not allowed", zap.String("filename", fileHeader.Filename))
		c.JSON(http.StatusBadRequest, gin.H{"error": "file extension is not allowed"})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		log.Error("Failed to open file", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to open file"})
		return
	}
	defer file.Close()

	taskID, err := h.imageService.UploadImage(c.Request.Context(), file, extension)
	if err != nil {
		log.Error("Image service failed to upload image", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start image processing"})
		return
	}
	log.Debug("Image uploaded successfully", zap.String("taskID", taskID))
	c.JSON(http.StatusOK, gin.H{"taskID": taskID})
}

func (h *ImageHandler) GetImage(c *gin.Context) {
	log := c.MustGet("logger").(*zap.Logger)
	log.Debug("Getting Image")
	taskID := c.Param("id")
	task, err := h.imageService.GetImage(c.Request.Context(), taskID)
	if err != nil {
		log.Error("Image service failed to get image", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get image"})
		return
	}
	log.Debug("Image retrieved successfully", zap.String("taskID", taskID))
	c.JSON(http.StatusOK, gin.H{"task": task})
}

func (h *ImageHandler) DeleteImage(c *gin.Context) {
	log := c.MustGet("logger").(*zap.Logger)
	log.Debug("Deleting Image")
	taskID := c.Param("id")
	if err := h.imageService.DeleteImage(c.Request.Context(), taskID); err != nil {
		// TODO: Добавить если не нашли id
		log.Error("Image service failed to delete image", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete image"})
		return
	}
	c.JSON(http.StatusNoContent, gin.H{})
}
