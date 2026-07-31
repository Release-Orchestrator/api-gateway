// Package main is the entry point for the API gateway.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Release-Orchestrator/api-gateway/internal/config"
	"github.com/Release-Orchestrator/api-gateway/internal/handler"
	"github.com/Release-Orchestrator/api-gateway/internal/middleware"
	"github.com/Release-Orchestrator/api-gateway/internal/proxy"
)

func main() {
	cfg := config.Load()

	userProxy, err := proxy.NewServiceProxy("user-service", cfg.UserServiceURL)
	if err != nil {
		log.Fatalf("failed to create user proxy: %v", err)
	}

	orderProxy, err := proxy.NewServiceProxy("order-service", cfg.OrderServiceURL)
	if err != nil {
		log.Fatalf("failed to create order proxy: %v", err)
	}

	paymentProxy, err := proxy.NewServiceProxy("payment-service", cfg.PaymentServiceURL)
	if err != nil {
		log.Fatalf("failed to create payment proxy: %v", err)
	}

	authProxy, err := proxy.NewServiceProxy("auth-service", cfg.AuthServiceURL)
	if err != nil {
		log.Fatalf("failed to create auth proxy: %v", err)
	}

	authValidator := middleware.NewAuthValidator(cfg.AuthServiceURL)

	r := gin.Default()

	healthHandler := handler.NewHealthHandler()
	healthHandler.RegisterRoutes(r)

	api := r.Group("/api/v1")
	{
		api.Any("/auth/*path", gin.WrapH(authProxy))

		protected := api.Group("")
		protected.Use(authValidator.Middleware())
		{
			protected.Any("/users/*path", gin.WrapH(userProxy))
			protected.Any("/orders/*path", gin.WrapH(orderProxy))
			protected.Any("/payments/*path", gin.WrapH(paymentProxy))
		}
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Printf("server starting on port %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("server forced to shutdown:", err)
	}

	log.Println("server exited")
}
