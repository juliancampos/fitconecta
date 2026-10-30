package httputil

import (
	"encoding/json"
	"net/http"
)

// Response padrao para respostas de sucesso
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
}

// ErrorResponse padrao para erros na API
type ErrorResponse struct {
	Success bool              `json:"success"`
	Error   string            `json:"error"`
	Details map[string]string `json:"details,omitempty"`
}

// JSON escreve uma resposta JSON com o status code especificado
func JSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// Success envia uma resposta HTTP 200/201 estruturada
func Success(w http.ResponseWriter, status int, data interface{}, message ...string) {
	msg := ""
	if len(message) > 0 {
		msg = message[0]
	}
	JSON(w, status, Response{
		Success: true,
		Data:    data,
		Message: msg,
	})
}

// Error envia uma resposta de erro padronizada
func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, ErrorResponse{
		Success: false,
		Error:   message,
	})
}

// ValidationError envia uma resposta de erro 422 com detalhes por campo
func ValidationError(w http.ResponseWriter, errors map[string]string) {
	JSON(w, http.StatusUnprocessableEntity, ErrorResponse{
		Success: false,
		Error:   "Dados informados sao invalidos",
		Details: errors,
	})
}
