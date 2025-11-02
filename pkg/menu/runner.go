package menu

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/mchmarny/momd/pkg/logger"
	"github.com/mchmarny/momd/pkg/server"
)

const (
	name = "momd"
)

// Run starts the menu server and blocks until the context is canceled or an error occurs.
// It automatically registers all menu item handlers and the root menu handler.
func (m *Menu) Run(ctx context.Context, opt ...server.Option) error {
	logger.SetDefaultLogger(name, m.Version)
	slog.Info("starting", "name", name)

	if opt == nil {
		opt = []server.Option{}
	}

	opt = append(opt,
		server.WithHandler("/", m.Handler()),
	)

	// Register all menu item handlers
	m.RegisterHandlers(func(pattern string, h http.Handler) {
		opt = append(opt, server.WithHandler(pattern, h))
	})

	// Create and run the server
	return server.New(opt...).Serve(ctx)
}
