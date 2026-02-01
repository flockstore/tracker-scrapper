package main

import (
	"log"

	"tracker-scrapper/internal/core/config"
	"tracker-scrapper/internal/core/logger"
	"tracker-scrapper/internal/core/server"
	orderadapter "tracker-scrapper/internal/features/orders/adapters"
	orderhandler "tracker-scrapper/internal/features/orders/handler"
	orderservice "tracker-scrapper/internal/features/orders/service"
	trackingadapter "tracker-scrapper/internal/features/tracking/adapters"
	trackinghandler "tracker-scrapper/internal/features/tracking/handler"
	"tracker-scrapper/internal/features/tracking/ports"
	trackingservice "tracker-scrapper/internal/features/tracking/service"

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

	// Initialize Order Adapter and run Health Check
	wcAdapter := orderadapter.NewWooCommerceAdapter(cfg.WooCommerce)
	if err := wcAdapter.HealthCheck(); err != nil {
		l.Fatal("WooCommerce Health Check Failed", zap.Error(err))
	}
	l.Info("WooCommerce connection verified")

	// Initialize Order Service & Handler
	orderService := orderservice.NewOrderService(wcAdapter)
	orderHandler := orderhandler.NewOrderHandler(orderService)

	// Initialize Tracking Providers
	coordinadoraAdapter := trackingadapter.NewCoordinadoraAdapter(cfg.Couriers.CoordinadoraURL)
	servientregaAdapter := trackingadapter.NewServientregaAdapter(cfg.Couriers.ServientregaURL)
	interrapidisimoAdapter := trackingadapter.NewInterrapidisimoAdapter(cfg.Couriers.InterrapidisimoURL)

	trackingProviders := []ports.TrackingProvider{
		coordinadoraAdapter,
		servientregaAdapter,
		interrapidisimoAdapter,
	}

	// Initialize Tracking Service & Handler
	trackingSvc := trackingservice.NewTrackingService(trackingProviders)
	trackingHdl := trackinghandler.NewTrackingHandler(trackingSvc)

	srv := server.New(cfg)

	// Register Routes
	srv.App.Get("/orders/:id", orderHandler.GetOrder)
	srv.App.Get("/tracking/:number", trackingHdl.GetTrackingHistory)

	if err := srv.Run(); err != nil {
		l.Fatal("Server failed to start", zap.Error(err))
	}
}
