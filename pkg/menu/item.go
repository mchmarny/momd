package menu

import "net/http"

// Item represents an individual item in the menu, which may contain sub-items.
type Item struct {
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
