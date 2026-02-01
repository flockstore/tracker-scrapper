package server

import (
	"fmt"

	"tracker-scrapper/internal/core/config"
	"tracker-scrapper/internal/core/logger"

	"github.com/gofiber/contrib/fiberzap/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/gofiber/swagger"
	"go.uber.org/zap"

	_ "tracker-scrapper/docs/swagger"
)

// Server holds the Fiber application and configuration.
type Server struct {
	// App is the main Fiber application instance.
	App *fiber.App
	// cfg holds the application configuration.
	cfg *config.AppConfig
}

// New creates a new Server instance with configured middleware.
func New(cfg *config.AppConfig) *Server {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
		AppName:               "tracker-scrapper",
	})

	app.Use(requestid.New(requestid.Config{
		Header: "X-Ray-ID",
	}))

	app.Use(fiberzap.New(fiberzap.Config{
		Logger: logger.Get(),
	}))

	app.Get("/swagger/*", swagger.HandlerDefault)

	return &Server{
		App: app,
		cfg: cfg,
	}
}

// Run starts the HTTP server.
func (s *Server) Run() error {
	addr := fmt.Sprintf(":%d", s.cfg.ServerPort)
	logger.Get().Info("Starting server", zap.String("address", addr))
	return s.App.Listen(addr)
}
