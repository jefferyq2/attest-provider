package handler

import (
	"net/http"
)

type readyHandler struct{}

// NewReadyHandler returns a readiness probe handler.
func NewReadyHandler() (http.Handler, error) {
	return &readyHandler{}, nil
}

func (h *readyHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
