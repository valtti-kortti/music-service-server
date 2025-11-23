package http_transport

import (
	"encoding/json"
	"mrs/internal/dto"
	"net/http"
)

func WriteJsonError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)

	_ = json.NewEncoder(w).Encode(dto.ErrorResponse{Message: message})
}
