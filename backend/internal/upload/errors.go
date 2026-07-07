package upload

import "errors"

var (
	ErrStorageUnavailable = errors.New("upload storage unavailable")
	ErrNoFile             = errors.New("file is required")
	ErrInvalidImage       = errors.New("invalid image file")
	ErrInvalidImageURL    = errors.New("invalid image url")
	ErrTooManyFiles       = errors.New("too many files")
	ErrFileTooLarge       = errors.New("file too large")
)
