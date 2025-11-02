package menu

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// Menu represents the root menu structure.
type Menu struct {
	// Title is the menu
	Title string `json:"title"`

	// Description of the menu
	Description string `json:"description,omitempty"`

	// Version of the menu
	Version string `json:"version,omitempty"`

	// Items is the list of menu items
	Items []Item `json:"items,omitempty"`
}

// ToJSON returns a JSON-serializable representation of the menu.
// The Handler field is excluded from serialization.
func (m *Menu) ToJSON() interface{} {
	return m
}

// RegisterHandlers walks through the menu tree and registers all handlers with the server.
// It recursively processes all menu items and their sub-items.
func (m *Menu) RegisterHandlers(register func(pattern string, handler http.Handler)) {
	for i := range m.Items {
		m.registerItem(&m.Items[i], register)
	}
}

// registerItem recursively registers a menu item and all its sub-items.
func (m *Menu) registerItem(item *Item, register func(pattern string, handler http.Handler)) {
	if item.Handler != nil {
		register(item.Path, item.Handler)
	}

	for i := range item.Items {
		m.registerItem(&item.Items[i], register)
	}
}

// Handler returns an HTTP handler that responds with the menu structure as JSON.
func (m *Menu) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info("handling menu request",
			"method", r.Method,
			"url", r.URL.Path,
		)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(m.ToJSON()); err != nil {
			slog.Error("failed to encode menu", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		slog.Info("menu response sent",
			"method", r.Method,
			"url", r.URL.Path,
			"status", http.StatusOK,
		)
	})
}
