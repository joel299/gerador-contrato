package handler

import (
	"errors"
	"net/http"

	"ia-infinito-contracts/internal/service"
)

type CNPJHandler struct {
	svc *service.CNPJService
}

func NewCNPJHandler(svc *service.CNPJService) *CNPJHandler {
	return &CNPJHandler{svc: svc}
}

func (h *CNPJHandler) Lookup(w http.ResponseWriter, r *http.Request) {
	cnpj := r.PathValue("cnpj")
	data, err := h.svc.Lookup(r.Context(), cnpj)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCNPJ):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		case errors.Is(err, service.ErrCNPJNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		case errors.Is(err, service.ErrCNPJAPIUnhealthy):
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "erro ao consultar CNPJ"})
		}
		return
	}
	writeJSON(w, http.StatusOK, data)
}
