package models

import (
	"github.com/wb-go/wbf/retry"
	"time"
)

const (
	ResizeToWidth     = 1024
	ResizeToHeight    = 768
	ThumbnailToWidth  = 128
	ThumbnailToHeight = 128

	ProcessPath  = "processed/%s/%s"
	OriginalPath = "original/%s%s"
	BasePath     = "images/"
)

var (
	AllowedExtensions = map[string]bool{
		".jpg": true,
		".png": true,
		".gif": true,
	}
	RetryStrategy = retry.Strategy{
		Attempts: 5,
		Delay:    time.Millisecond,
		Backoff:  2,
	}
	RequestedOperations = []string{"resize", "thumbnail", "watermark"}
)

type TaskStatus string

const (
	StatusFailed     TaskStatus = "FAILED"
	StatusProcessing TaskStatus = "PROCESSING"
	StatusComplete   TaskStatus = "COMPLETE"
)

type Task struct {
	ID                  string     `json:"id"`
	Status              TaskStatus `json:"status"`
	OriginalPath        string     `json:"original_path"`
	RequestedOperations []string   `json:"requested_operations"`
	CreatedAt           time.Time  `json:"created_at"`
}

type ProcessingCommand struct {
	ID                  string    `json:"id"`
	OriginalPath        string    `json:"original_path"`
	RequestedOperations []string  `json:"requested_operations"`
	CreatedAt           time.Time `json:"created_at"`
}
