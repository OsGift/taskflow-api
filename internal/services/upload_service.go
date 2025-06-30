package services

import (
	"context"
	"mime/multipart"
	"fmt"
	"time"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

// UploadService handles file uploads to Cloudinary
type UploadService struct {
	cld    *cloudinary.Cloudinary
	ctx    context.Context
}

// NewUploadService creates a new UploadService instance
func NewUploadService(cloudName, apiKey, apiSecret string) *UploadService {
	cld, err := cloudinary.NewFromParams(cloudName, apiKey, apiSecret)
	if err != nil {
		// In a real application, you'd log this fatal error or return it.
		// For this example, we'll panic if Cloudinary credentials are bad.
		panic(fmt.Sprintf("Failed to initialize Cloudinary: %v", err))
	}
	return &UploadService{
		cld: cld,
		ctx: context.Background(), // Using a background context for the service,
	}
}

// UploadFile uploads a file to Cloudinary and returns its URL
func (s *UploadService) UploadFile(fileHeader *multipart.FileHeader) (string, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Upload parameters, can be customized
	uploadResult, err := s.cld.Upload.Upload(s.ctx, file, uploader.UploadParams{
		Folder: "taskflow-uploads", // Optional: organize uploads in a specific folder
		PublicID: fmt.Sprintf("%s_%d", fileHeader.Filename, time.Now().UnixNano()), // Unique public ID
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file to Cloudinary: %w", err)
	}

	return uploadResult.SecureURL, nil
}
