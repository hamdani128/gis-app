package response

import (
	"encoding/json"
	"log"
	"net/http"
)

type APIResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

func JSON(
	w http.ResponseWriter,
	status int,
	message string,
	data any,
) {
	result := APIResponse{
		Status:  status,
		Message: message,
		Data:    data,
	}

	// Encode dahulu agar kegagalan JSON bisa menghasilkan HTTP 500.
	body, err := json.Marshal(result)
	if err != nil {
		log.Printf("Gagal encode respons: %v", err)

		status = http.StatusInternalServerError
		body = []byte(
			`{"status":500,"message":"Gagal membuat respons","data":null}`,
		)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if _, err := w.Write(body); err != nil {
		log.Printf("Gagal mengirim respons: %v", err)
	}
}

func Success(w http.ResponseWriter, message string, data any) {
	JSON(w, http.StatusOK, message, data)
}

func Created(w http.ResponseWriter, message string, data any) {
	JSON(w, http.StatusCreated, message, data)
}

func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, message, nil)
}
