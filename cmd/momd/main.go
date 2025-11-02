package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/mchmarny/momd/pkg/menu"
	"github.com/mchmarny/momd/pkg/server"
)

var (
	version = "v0.0.0" // Set at build time via -ldflags "-X main.version=version"

	port = flag.Int("port", server.DefaultPort, "Port to run the server on")
)

func main() {
	// Parse command-line flags
	flag.Parse()

	// Build the menu and its items
	m := makeMenu()

	ctx := context.Background()

	// Run the menu server
	if err := m.Run(ctx, server.WithPort(*port)); err != nil {
		slog.Error("server error", "error", err)
	}
}

// makeMenu constructs the menu structure with items and sub-items.
// The handles and paths are set up for each menu item and point to your own handles.
func makeMenu() *menu.Menu {
	return &menu.Menu{
		Title:       fmt.Sprintf("Root Menu (v%s)", version),
		Description: "This is the root menu",
		Version:     version,
		Items: []menu.Item{
			{
				Title:       "Item 1",
				Description: "This is item 1",
				Path:        "/item1",
				Handler:     simple(),
			},
			{
				Title:       "Item 2",
				Description: "This is item 2",
				Path:        "/item2",
				Handler:     simple(),
				Items: []menu.Item{
					{
						Title:       "Subitem 1",
						Description: "This is subitem 1",
						Path:        "/item2/subitem1",
						Handler:     simple(),
					},
					{
						Title:       "Subitem 2",
						Description: "This is subitem 2",
						Path:        "/item2/subitem2",
						Handler:     simple(),
					},
				},
			},
		},
	}
}

// simple returns a simple handler that responds with request information.
func simple() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info("handling",
			"method", r.Method,
			"url", r.URL.Path,
		)

		data := map[string]interface{}{
			"message": "Hello, World!",
			"method":  r.Method,
			"url":     r.URL.Path,
		}

		writeJSON(w, http.StatusOK, data)

		slog.Info("completed",
			"method", r.Method,
			"url", r.URL.Path,
			"status", http.StatusOK,
		)
	})
}

func writeError(w http.ResponseWriter, status int, message string) {
	slog.Error("handling error response",
		"status", status,
		"message", message,
	)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	fmt.Fprintf(w, `{"error": "%s"}`, message)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	slog.Info("handling JSON response",
		"status", status,
	)

	jsonData, err := json.Marshal(data)
	if err != nil {
		slog.Error("failed to marshal JSON response", "error", err)
		writeError(w, http.StatusInternalServerError, "error, see logs for details")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(jsonData); err != nil {
		slog.Error("failed to write JSON response", "error", err)
		http.Error(w, fmt.Sprintf("internal server error: %v", err), http.StatusInternalServerError)
		return
	}
	slog.Info("json response sent successfully", "status", status)
}
