package httpapi

const (
	CodeInvalidRequest  Code = 400001
	CodeIDRequired      Code = 400002
	CodeUnauthorized    Code = 401001
	CodeInvalidAPIKey   Code = 403001
	CodeForbidden       Code = 403002
	CodeTooManyRequests Code = 429001
	CodeInternal        Code = 500001

	CodeUserIDRequired          Code = 400101
	CodeUserSnapshotInvalid     Code = 400102
	CodeUserVersionInvalid      Code = 400103
	CodeUserStatusInvalid       Code = 400104
	CodeUserFieldTooLong        Code = 400105
	CodeUserNotFound            Code = 404101
	CodeUserVersionStale        Code = 409101
	CodeUserSnapshotWriteFailed Code = 500101
)
