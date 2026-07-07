package upload

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// UploadImage 单图上传，表单字段 file
func (h *Handler) UploadImage(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		BadRequest(c, "file is required")
		return
	}

	res, err := h.svc.UploadImage(c.Request.Context(), file)
	if err != nil {
		writeUploadError(c, err)
		return
	}
	Success(c, "success", res)
}

// UploadImages 批量图片上传，表单字段 files；兼容单文件字段 file。
func (h *Handler) UploadImages(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		writeUploadError(c, ErrNoFile)
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		files = form.File["file"]
	}

	res, err := h.svc.UploadImages(c.Request.Context(), files)
	if err != nil {
		writeUploadError(c, err)
		return
	}
	Success(c, "success", res)
}

func writeUploadError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNoFile):
		BadRequest(c, err.Error())
	case errors.Is(err, ErrStorageUnavailable):
		Error(c, http.StatusServiceUnavailable, http.StatusServiceUnavailable, err.Error())
	case errors.Is(err, ErrInvalidImage):
		BadRequest(c, err.Error())
	case errors.Is(err, ErrInvalidImageURL):
		BadRequest(c, err.Error())
	case errors.Is(err, ErrTooManyFiles):
		BadRequest(c, err.Error())
	case errors.Is(err, ErrFileTooLarge):
		BadRequest(c, err.Error())
	default:
		InternalServerError(c, err.Error())
	}
}
