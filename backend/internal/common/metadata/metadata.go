package metadata

import "context"

type contextKey string

const (
	// 请求元数据键
	KeyTraceID contextKey = "trace_id"
	KeyUID     contextKey = "uid"
)

// WithTraceID 注入 TraceID
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, KeyTraceID, traceID)
}

// GetTraceID 提取 TraceID
func GetTraceID(ctx context.Context) string {
	if v, ok := ctx.Value(KeyTraceID).(string); ok {
		return v
	}
	return ""
}
