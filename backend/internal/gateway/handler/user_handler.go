package handler

import (
	"context"
	"errors"
	"net/http"

	"d-im/internal/gateway/handler/middleware"
	"d-im/pkg/model"

	"go.mongodb.org/mongo-driver/mongo"
)

type userReader interface {
	FindByID(ctx context.Context, id string) (*model.User, error)
}

// UserHandler 用户 HTTP 处理器
type UserHandler struct {
	users userReader
}

// NewUserHandler 创建用户处理器
func NewUserHandler(users userReader) *UserHandler {
	return &UserHandler{users: users}
}

// GetMe 获取当前登录用户
func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	id := middleware.GetUserID(r.Context())
	if id == "" {
		writeAPIError(w, http.StatusUnauthorized, 401001, "unauthorized")
		return
	}

	h.writeUserByID(w, r, id)
}

// GetUser 获取指定用户详情
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeAPIError(w, http.StatusBadRequest, 400002, "id is required")
		return
	}

	h.writeUserByID(w, r, id)
}

func (h *UserHandler) writeUserByID(w http.ResponseWriter, r *http.Request, id string) {
	user, err := h.users.FindByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			writeAPIError(w, http.StatusNotFound, 404001, "user not found")
			return
		}
		writeAPIError(w, http.StatusInternalServerError, 500101, "get user failed")
		return
	}

	writeAPISuccess(w, userDTOFromModel(user))
}

type userDTO struct {
	ID       string `json:"id"`
	Nickname string `json:"nickname,omitempty"`
	Avatar   string `json:"avatar,omitempty"`
	Status   string `json:"status,omitempty"`
}

func userDTOFromModel(user *model.User) userDTO {
	return userDTO{
		ID:       user.ID,
		Nickname: user.Nickname,
		Avatar:   user.Avatar,
		Status:   user.Status,
	}
}
