package menu

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

const (
	// ItemTypeButton calls back to the server.
	ItemTypeCallback ItemType = "callback"
	// ItemTypeLink opens a link using default handler.
	ItemTypeLink ItemType = "link"
)

// ItemType represents the type of a menu item.
// Dictates what happens on user click.
type ItemType string

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

// Item represents an individual item in the menu, which may contain sub-items.
type Item struct {
	// Type indicates the type of the menu item (e.g., callback, link).
	Type ItemType `json:"type"`

	// Path is the unique identifier for the menu item within the scope of its parent.
	Path string `json:"path"`

	// Handler is the HTTP handler associated with this menu item.
	// This field is not serialized to JSON.
	Handler http.Handler `json:"-"`

	// Title is the title of the menu item.
	Title string `json:"title"`

	// Description is an optional description of the menu item.
	Description string `json:"description,omitempty"`

	// Items are the sub-items of this menu item.
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
