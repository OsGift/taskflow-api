package handlers

import (
	"fmt"
	"net/http"

	"github.com/OsGift/taskflow-api/internal/services"
	"github.com/OsGift/taskflow-api/internal/utils"
)

// UploadHandler handles file upload related HTTP requests
type UploadHandler struct {
	uploadService *services.UploadService
}

// NewUploadHandler creates a new UploadHandler
func NewUploadHandler(us *services.UploadService) *UploadHandler {
	return &UploadHandler{
		uploadService: us,
	}
}

// UploadFile handles file uploads to Cloudinary
func (h *UploadHandler) UploadFile(w http.ResponseWriter, r *http.Request) {
	// Permission check is done by middleware (e.g., any logged-in user can upload their profile pic)

	// Max 10MB file size
	r.ParseMultipartForm(10 << 20) // 10MB

	file, fileHeader, err := r.FormFile("file") // "file" is the name of the form field
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, fmt.Sprintf("Error retrieving file from form: %v", err))
		return
	}
	defer file.Close()

	if fileHeader.Size == 0 {
		utils.RespondWithError(w, http.StatusBadRequest, "Uploaded file is empty.")
		return
	}

	// You might want to add file type validation here (e.g., only images)
	// if !strings.HasPrefix(fileHeader.Header.Get("Content-Type"), "image/") {
	// 	utils.RespondWithError(w, http.StatusBadRequest, "Only image files are allowed.")
	// 	return
	// }

	imageURL, err := h.uploadService.UploadFile(fileHeader)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to upload file: %v", err))
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "File uploaded successfully", "url": imageURL})
}
