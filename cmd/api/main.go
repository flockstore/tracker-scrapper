package main

import (
	"log"

	"tracker-scrapper/internal/core/config"
	"tracker-scrapper/internal/core/logger"
	"tracker-scrapper/internal/core/server"
	adapter "tracker-scrapper/internal/features/orders/adapters"
	"tracker-scrapper/internal/features/orders/handler"
	"tracker-scrapper/internal/features/orders/service"

	"go.uber.org/zap"
)

// @title Tracker Scrapper API
// @version 1.0
// @description This API provides order tracking functionality by integrating with WooCommerce.
// @contact.name API Support
// @contact.email support@trackerscrapper.com
// @license.name MIT
// @host localhost:8080
// @BasePath /
func main() {
	cfg, err := config.Load(".")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if err := logger.Init(cfg.Environment, cfg.LogLevel); err != nil {
		log.Fatalf("Failed to init logger: %v", err)
	}
	defer logger.Sync()

	l := logger.Get()
	l.Info("Application starting",
		zap.String("environment", cfg.Environment),
		zap.String("log_level", cfg.LogLevel),
	)

	// Initialize Adapter and run Health Check
	wcAdapter := adapter.NewWooCommerceAdapter(cfg.WooCommerce)
	if err := wcAdapter.HealthCheck(); err != nil {
		l.Fatal("WooCommerce Health Check Failed", zap.Error(err))
	}
	l.Info("WooCommerce connection verified")

	// Initialize Service & Handler
	orderService := service.NewOrderService(wcAdapter)
	orderHandler := handler.NewOrderHandler(orderService)

	srv := server.New(cfg)

	// Register Routes
	srv.App.Get("/orders/:id", orderHandler.GetOrder)

	if err := srv.Run(); err != nil {
		l.Fatal("Server failed to start", zap.Error(err))
	}
}
