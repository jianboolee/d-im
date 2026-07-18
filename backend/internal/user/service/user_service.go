package service

import (
	"context"

	"d-im/internal/user/repository"
	"d-im/pkg/model"
)

// UserService 系统内部用户查询服务。
// 用户数据由业务系统通过 SDK HTTP 接口同步。
type UserService struct {
	repo *repository.UserRepo
}

// NewUserService 创建用户服务
func NewUserService(repo *repository.UserRepo) *UserService {
	return &UserService{repo: repo}
}

// FindByID 查询用户
func (s *UserService) FindByID(ctx context.Context, id string) (*model.User, error) {
	return s.repo.FindByID(ctx, id)
}
