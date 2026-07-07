package service

import "errors"

var (
	ErrForbidden      = errors.New("forbidden")
	ErrInvalid        = errors.New("invalid group operation")
	ErrGroupFull      = errors.New("group is full")
	ErrOwnerRequired  = errors.New("owner required")
	ErrMemberNotFound = errors.New("member not found")
)
