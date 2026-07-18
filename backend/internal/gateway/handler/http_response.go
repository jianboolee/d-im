package handler

import (
	"net/http"

	"d-im/internal/gateway/httpapi"
)

func writeSuccess(w http.ResponseWriter, data interface{}) {
	httpapi.WriteSuccess(w, http.StatusOK, data)
}

func writeError(w http.ResponseWriter, status int, code int, message string) {
	httpapi.WriteError(w, status, httpapi.Error{Code: httpapi.Code(code), Message: message})
}
