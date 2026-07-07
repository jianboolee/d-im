package handler

import (
	"errors"
	"net/http"

	"d-im/internal/gateway/handler/middleware"
	mediaSvc "d-im/internal/media/service"
)

type UploadHandler struct {
	uploads *mediaSvc.UploadService
}

func NewUploadHandler(uploads *mediaSvc.UploadService) *UploadHandler {
	return &UploadHandler{uploads: uploads}
}

// UploadImage 上传图片。
// POST /api/v1/uploads/image
func (h *UploadHandler) UploadImage(w http.ResponseWriter, r *http.Request) {
	if middleware.GetUserID(r.Context()) == "" {
		writeAPIError(w, http.StatusUnauthorized, 401001, "unauthorized")
		return
	}
	if h.uploads == nil {
		writeAPIError(w, http.StatusInternalServerError, 500501, "upload service is unavailable")
		return
	}

	if err := r.ParseMultipartForm(h.uploads.MaxImageSize()); err != nil {
		writeAPIError(w, http.StatusBadRequest, 400030, "invalid multipart form")
		return
	}
	file, _, err := r.FormFile("file")
	if err == nil {
		_ = file.Close()
	}
	if errors.Is(err, http.ErrMissingFile) {
		writeAPIError(w, http.StatusBadRequest, 400031, "file is required")
		return
	}

	header := r.MultipartForm.File["file"]
	if len(header) == 0 {
		writeAPIError(w, http.StatusBadRequest, 400031, "file is required")
		return
	}

	uploaded, err := h.uploads.UploadImage(r.Context(), header[0])
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, 400032, err.Error())
		return
	}
	writeAPISuccess(w, uploaded)
}
