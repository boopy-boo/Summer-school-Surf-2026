package handler

import (
	"encoding/json"
	"net/http"

	"pottery-api/internal/domain"
	"pottery-api/internal/service"
)

type AuthHandler struct {
	svc *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

type otpSendRequest struct {
	Phone string `json:"phone" validate:"required,e164"`
}

type otpVerifyRequest struct {
	Phone string `json:"phone" validate:"required,e164"`
	Code  string `json:"code" validate:"required,len=6,numeric"`
	Name  string `json:"name" validate:"required,min=1,max=100"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

func (h *AuthHandler) SendOTP(w http.ResponseWriter, r *http.Request) {
	var req otpSendRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request")
		return
	}

	ttl, err := h.svc.SendOTP(r.Context(), req.Phone)
	if err != nil {
		if err == domain.ErrTooManyRequests {
			respondError(w, http.StatusTooManyRequests, "too many requests")
			return
		}
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{"ttl_seconds": ttl})
}

func (h *AuthHandler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var req otpVerifyRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request")
		return
	}

	tokens, profile, err := h.svc.VerifyOTP(r.Context(), req.Phone, req.Code, req.Name)
	if err != nil {
		switch {
		case err == domain.ErrInvalidOTP:
			respondError(w, http.StatusBadRequest, "invalid otp")
		case err == domain.ErrTooManyRequests:
			respondError(w, http.StatusTooManyRequests, "too many attempts")
		default:
			respondError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"tokens": tokens,
		"user":   profile,
	})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request")
		return
	}

	tokens, err := h.svc.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}

	respondJSON(w, http.StatusOK, tokens)
}