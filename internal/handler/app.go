package handler

import (
	"log/slog"

	"github.com/adocoder12/chatRoom/internal/config"
)

type Application struct {
	logger  *slog.Logger
	config  *config.Config
	version string
}

func NewApplication(logger *slog.Logger, config *config.Config, version string) *Application {
	return &Application{
		logger:  logger,
		config:  config,
		version: version,
	}
}
