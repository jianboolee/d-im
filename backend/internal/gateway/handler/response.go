package handler

import (
	"encoding/json"
	"net/http"
)

type apiResponse struct {
	Code  int         `json:"code"`
	Data  interface{} `json:"data"`
	Error string      `json:"error"`
}

func writeAPISuccess(w http.ResponseWriter, data interface{}) {
	if data == nil {
		data = map[string]interface{}{}
	}
	writeAPIResponse(w, http.StatusOK, apiResponse{
		Code:  0,
		Data:  data,
		Error: "",
	})
}

func writeAPIError(w http.ResponseWriter, status int, code int, message string) {
	writeAPIResponse(w, status, apiResponse{
		Code:  code,
		Data:  nil,
		Error: message,
	})
}

func writeAPIResponse(w http.ResponseWriter, status int, resp apiResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}
