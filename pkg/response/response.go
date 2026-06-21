package response

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/friend-diaries/backend/pkg/apperror"
)

type envelope struct {
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(envelope{Data: data}); err != nil {
		slog.Error("encoding response", "error", err)
	}
}

func Created(w http.ResponseWriter, data any) {
	JSON(w, http.StatusCreated, data)
}

func OK(w http.ResponseWriter, data any) {
	JSON(w, http.StatusOK, data)
}

func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func Error(w http.ResponseWriter, err error) {
	if appErr, ok := apperror.As(err); ok {
		if appErr.Code >= 500 {
			slog.Error(appErr.Message, "error", appErr.Err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(appErr.Code)
		_ = json.NewEncoder(w).Encode(envelope{Error: appErr.Message})
		return
	}

	slog.Error("unexpected error", "error", err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	_ = json.NewEncoder(w).Encode(envelope{Error: "internal server error"})
}
