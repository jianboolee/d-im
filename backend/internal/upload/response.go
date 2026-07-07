package upload

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const SuccessCode = 0

// Response 统一响应结构
type ResponseData struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// PaginationResponse 分页响应结构
type PaginationResponseData struct {
	Code    int            `json:"code"`
	Message string         `json:"message"`
	Data    interface{}    `json:"data"`
	Meta    PaginationMeta `json:"meta"`
}

// PaginationMeta 分页元数据
type PaginationMeta struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalPages int   `json:"total_pages"`
}

// SuccessWithMessage 带消息的成功响应
func Success(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, ResponseData{
		Code:    SuccessCode,
		Message: message,
		Data:    data,
	})
}

// ErrorResponse 带HTTP状态码的错误响应
func Error(c *gin.Context, httpStatus, code int, message string) {
	c.JSON(httpStatus, ResponseData{
		Code:    code,
		Message: message,
	})
}

// PaginationResponse 分页响应
func Pagination(c *gin.Context, data interface{}, total int64, page, pageSize int) {
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	c.JSON(http.StatusOK, PaginationResponseData{
		Code:    SuccessCode,
		Message: "success",
		Data:    data,
		Meta: PaginationMeta{
			Total:      total,
			Page:       page,
			PageSize:   pageSize,
			TotalPages: totalPages,
		},
	})
}

// BadRequest 400错误
func BadRequest(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, http.StatusBadRequest, message)
}

// Unauthorized 401错误
func Unauthorized(c *gin.Context, message string) {
	Error(c, http.StatusUnauthorized, http.StatusUnauthorized, message)
}

// Forbidden 403错误
func Forbidden(c *gin.Context, message string) {
	Error(c, http.StatusForbidden, http.StatusForbidden, message)
}

// NotFound 404错误
func NotFound(c *gin.Context, message string) {
	Error(c, http.StatusNotFound, http.StatusNotFound, message)
}

// InternalServerError 500错误
func InternalServerError(c *gin.Context, message string) {
	Error(c, http.StatusInternalServerError, http.StatusInternalServerError, message)
}
