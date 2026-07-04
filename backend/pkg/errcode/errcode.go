package errcode

// 通用错误码
const (
	ErrCodeOK           = 0
	ErrCodeInternal     = 500
	ErrCodeBadRequest   = 400
	ErrCodeUnauthorized = 401
	ErrCodeNotFound     = 404
	ErrCodeTooManyReq   = 429
)

// 消息错误码 (1000-1999)
const (
	ErrCodeMessageSendFailed = 1001
	ErrCodeMessageNotFound   = 1002
	ErrCodeMessageRecalled   = 1003
	ErrCodeRecallForbidden   = 1004
	ErrCodeContentInvalid    = 1005
)

// 会话错误码 (2000-2999)
const (
	ErrCodeConvNotFound   = 2001
	ErrCodeConvCreateFail = 2002
)

// 群组错误码 (3000-3999)
const (
	ErrCodeGroupNotFound   = 3001
	ErrCodeGroupCreateFail = 3002
	ErrCodeMemberAddFail   = 3003
)
