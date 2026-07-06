// @title Гончарная мастерская API
// @version 1.0.0
package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"pottery-api/internal/config"
	"pottery-api/internal/handler"
	httpadapter "pottery-api/internal/adapter/http"
	redisadapter "pottery-api/internal/adapter/redis"
	appmiddleware "pottery-api/internal/handler/middleware"
	httpserver "pottery-api/internal/http"
	"pottery-api/internal/service"
	"pottery-api/internal/service/auth"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// Adapters
	redisClient := redisadapter.NewOTPStore(cfg.RedisAddr, cfg.RedisPass)
	backendClient := httpadapter.NewBackendClient(cfg.BackendURL, cfg.BackendTimeout)

	// JWT
	jwtService := auth.NewJWTService(cfg.JWTSecret)

	// Services
	authSvc := service.NewAuthService(redisClient, backendClient, jwtService)
	slotSvc := service.NewSlotService(backendClient)
	bookingSvc := service.NewBookingService(backendClient, backendClient)

	// Handlers
	authH := handler.NewAuthHandler(authSvc)
	slotH := handler.NewSlotHandler(slotSvc)
	bookingH := handler.NewBookingHandler(bookingSvc)

	// Router
	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Route("/auth", func(r chi.Router) {
		r.Post("/otp/send", authH.SendOTP)
		r.Post("/otp/verify", authH.VerifyOTP)
		r.Post("/refresh", authH.Refresh)
	})

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(appmiddleware.Auth(jwtService))
		r.Get("/slots", slotH.List)
		r.Get("/slots/{slotId}", slotH.GetByID)
		r.Get("/masters", slotH.ListMasters)

		r.Post("/bookings", bookingH.Create)
		r.Get("/bookings", bookingH.List)
		r.Get("/bookings/{bookingId}", bookingH.GetByID)
		r.Delete("/bookings/{bookingId}", bookingH.Cancel)
	})

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger, _ := zap.NewDevelopment()
	srv := httpserver.New(r, cfg.HTTPAddr, logger)
	log.Printf("starting server on %s", cfg.HTTPAddr)
	if err := srv.Run(ctx); err != nil {
		log.Fatalf("server error: %v", err)
	}
}