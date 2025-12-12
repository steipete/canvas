package rpc

import (
	"net/http"
	"strings"
)

type Handler struct {
	Token string
	Mux   *http.ServeMux
}

func NewHandler(token string) *Handler {
	return &Handler{
		Token: token,
		Mux:   http.NewServeMux(),
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.Token != "" {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") || strings.TrimPrefix(auth, "Bearer ") != h.Token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	}
	h.Mux.ServeHTTP(w, r)
}
