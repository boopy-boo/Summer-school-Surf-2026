package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"pottery-api/internal/domain"
	"pottery-api/internal/service"
)

type SlotHandler struct {
	svc *service.SlotService
}

func NewSlotHandler(svc *service.SlotService) *SlotHandler {
	return &SlotHandler{svc: svc}
}

func (h *SlotHandler) List(w http.ResponseWriter, r *http.Request) {
	var filter domain.SlotFilter

	if v := r.URL.Query().Get("limit"); v != "" {
		filter.Limit, _ = strconv.Atoi(v)
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		filter.Offset, _ = strconv.Atoi(v)
	}

	if v := r.URL.Query().Get("date_from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filter.DateFrom = &t
		}
	}
	if v := r.URL.Query().Get("date_to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filter.DateTo = &t
		}
	}

	filter.ProgramTypes = r.URL.Query()["program_type"]
	filter.MasterIDs = r.URL.Query()["master_id"]
	filter.OnlyAvailable = r.URL.Query().Get("only_available") == "true"

	if filter.Limit == 0 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}

	items, total, err := h.svc.ListSlots(r.Context(), filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	respondJSON(w, http.StatusOK, domain.SlotListResponse{Items: items, Total: total})
}

func (h *SlotHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "slotId")
	slot, err := h.svc.GetSlot(r.Context(), id)
	if err != nil {
		if err == domain.ErrNotFound {
			respondError(w, http.StatusNotFound, "slot not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	respondJSON(w, http.StatusOK, slot)
}

func (h *SlotHandler) ListMasters(w http.ResponseWriter, r *http.Request) {
	masters, err := h.svc.ListMasters(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	respondJSON(w, http.StatusOK, masters)
}