package httpapi

import (
	"encoding/json"
	"log"
	"net/http"
)

// Code 是对外稳定的 API 业务错误码。
type Code int

const CodeOK Code = 0

// Response 是 gateway JSON API 的统一响应封装。
type Response struct {
	Code  Code   `json:"code"`
	Data  any    `json:"data"`
	Error string `json:"error"`
}

// Error 描述一个可对外返回的 API 错误。
type Error struct {
	Code    Code
	Message string
}

// WriteSuccess 写入成功响应。
func WriteSuccess(w http.ResponseWriter, status int, data any) {
	if data == nil {
		data = map[string]any{}
	}
	writeJSON(w, status, Response{Code: CodeOK, Data: data, Error: ""})
}

// WriteError 写入错误响应。
func WriteError(w http.ResponseWriter, status int, apiError Error) {
	writeJSON(w, status, Response{Code: apiError.Code, Data: nil, Error: apiError.Message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("[httpapi] encode response failed: %v", err)
	}
}
