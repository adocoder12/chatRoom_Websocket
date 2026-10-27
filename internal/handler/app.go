package handler

import (
	"log/slog"

	"github.com/adocoder12/chatRoom/internal/chat"
	"github.com/adocoder12/chatRoom/internal/config"
)

type Application struct {
	logger  *slog.Logger
	config  *config.Config
	version string
	hub *chat.Hub
}

func NewApplication(logger *slog.Logger, config *config.Config, version string,	hub *chat.Hub) *Application {
	return &Application{
		logger:  logger,
		config:  config,
		version: version,
		hub: hub,
	}
}
