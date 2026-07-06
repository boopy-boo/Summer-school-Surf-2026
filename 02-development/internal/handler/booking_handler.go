package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"pottery-api/internal/domain"
	"pottery-api/internal/handler/middleware"
	"pottery-api/internal/service"
)

type BookingHandler struct {
	svc *service.BookingService
}

func NewBookingHandler(svc *service.BookingService) *BookingHandler {
	return &BookingHandler{svc: svc}
}

func (h *BookingHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req domain.BookingCreateRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request")
		return
	}

	clientID := middleware.ClientID(r.Context())
	booking, err := h.svc.Create(r.Context(), clientID, req)
	if err != nil {
		switch {
		case err == domain.ErrGone:
			respondError(w, http.StatusGone, "slot cancelled by workshop")
		case err == domain.ErrConflict:
			respondError(w, http.StatusConflict, "no seats available")
		default:
			respondError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}
	respondJSON(w, http.StatusCreated, booking)
}

func (h *BookingHandler) List(w http.ResponseWriter, r *http.Request) {
	limit := 20
	offset := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	if limit > 100 {
		limit = 100
	}

	clientID := middleware.ClientID(r.Context())
	items, total, err := h.svc.List(r.Context(), clientID, limit, offset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	respondJSON(w, http.StatusOK, domain.BookingListResponse{Items: items, Total: total})
}

func (h *BookingHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "bookingId")
	clientID := middleware.ClientID(r.Context())
	booking, err := h.svc.GetByID(r.Context(), clientID, id)
	if err != nil {
		if err == domain.ErrNotFound {
			respondError(w, http.StatusNotFound, "booking not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	respondJSON(w, http.StatusOK, booking)
}

func (h *BookingHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "bookingId")
	clientID := middleware.ClientID(r.Context())
	if err := h.svc.Cancel(r.Context(), clientID, id); err != nil {
		if err == domain.ErrNotFound {
			respondError(w, http.StatusNotFound, "booking not found")
			return
		}
		if err == domain.ErrBadRequest {
			respondError(w, http.StatusBadRequest, "cannot cancel")
			return
		}
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}