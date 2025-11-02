package menu

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/mchmarny/momd/pkg/logger"
	"github.com/mchmarny/momd/pkg/server"
)

var (
	version = "dev"     // Set at build time via -ldflags "-X main.version=version"
	commit  = "none"    // Set at build time via -ldflags "-X main.commit=commit"
	date    = "unknown" // Set at build time via -ldflags "-X main.date=date"
)

// Run starts the menu server and blocks until the context is canceled or an error occurs.
// It automatically registers all menu item handlers and the root menu handler.
func (m *Menu) Run(ctx context.Context, opt ...server.Option) error {
	logger.SetDefaultStructuredLogger("momd", version)
	slog.Info("starting momd", "commit", commit, "date", date)

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
