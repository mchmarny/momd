# momd - Mac OS Menu App with Go Back-end

A Go-based menu server with a native macOS menu bar wrapper.

## Quick Start

### Build and Run

```bash
# Build and run the macOS app (includes Go server)
make run

# Or just build without running
make app
```

### What Happens

1. A menu bar icon appears in your macOS menu bar (looks like a list icon)
2. The Go server starts automatically in the background
3. Click the menu bar icon to see your menu
4. Click any menu item to trigger its server-side handler

## Project Structure

```
momd/
├── cmd/momd/main.go          # Go server entry point & menu definition
├── pkg/
│   ├── menu/                 # Reusable menu package
│   ├── server/               # HTTP server package
│   ├── logger/               # Logging utilities
│   └── metric/               # Metrics utilities
├── macos/
│   ├── momd/
│   │   ├── main.swift        # Swift app entry point
│   │   ├── AppDelegate.swift # Menu bar app logic
│   │   └── Info.plist        # App metadata
│   ├── build/                # Build output
│   └── README.md            # macOS-specific docs
└── Makefile                 # Build automation
```

## Customizing Your Menu

Edit `cmd/momd/main.go` and modify the `makeMenu()` function:

```go
func makeMenu() *menu.Menu {
    return &menu.Menu{
        Title:       "My App",
        Description: "My custom menu",
        Items: []menu.Item{
            {
                Title:       "Say Hello",
                Description: "Prints a greeting (hover text)",
                Type:        menu.ItemTypeCallback,  // Calls server
                OnClick:     "/hello",
                Shortcut:    "cmd+h",               // ⌘H
                Handler:     myHandler(),
            },
            {
                Title:       "Open GitHub",
                Description: "Opens GitHub in browser",
                Type:        menu.ItemTypeLink,      // Opens URL
                OnClick:     "https://github.com",
                Shortcut:    "cmd+g",                // ⌘G
            },
            {
                Title:       "Submenu",
                Description: "A submenu example",
                Items: []menu.Item{
                    {
                        Title:       "Nested Item",
                        Description: "A nested callback",
                        Type:        menu.ItemTypeCallback,
                        OnClick:     "/submenu/nested",
                        Shortcut:    "cmd+shift+n",  // ⌘⇧N
                        Handler:     myHandler(),
                    },
                },
            },
        },
    }
}
```

### Menu Item Types

There are two types of menu items:

- **`menu.ItemTypeCallback`**: Calls back to the Go server when clicked
  - Requires a `Handler` and `OnClick` path (e.g., `"/hello"`)
  - Server handles the request and returns a response
  
- **`menu.ItemTypeLink`**: Opens a URL using the system default handler
  - Requires only `OnClick` with a full URL (e.g., `"https://github.com"`)
  - Opens in default browser, mail client, etc. depending on URL scheme
  - No server-side handler needed

### Menu Item Fields

- **`Title`**: The text displayed in the menu (required)
- **`Description`**: Tooltip text shown on hover (optional)
- **`Type`**: Either `menu.ItemTypeCallback` or `menu.ItemTypeLink`
- **`OnClick`**: 
  - For callbacks: server path like `"/action"`
  - For links: full URL like `"https://example.com"` or `"mailto:user@example.com"`
- **`Shortcut`**: Keyboard shortcut (optional)
  - Format: `"cmd+key"`, `"cmd+shift+key"`, etc.
  - Modifiers: `cmd`, `ctrl`, `opt`/`option`/`alt`, `shift`
  - Examples: `"cmd+1"`, `"cmd+shift+g"`, `"ctrl+opt+d"`
- **`Handler`**: HTTP handler function (only for callback types)
- **`Items`**: Nested submenu items (optional)

## Available Make Targets

```bash
make run        # Build and run macOS app
make app        # Build macOS app (builds Go server first)
make server     # Run Go server directly (for testing)
make build      # Build Go server binary only
make clean      # Clean macOS and Go build artifacts
make test       # Run tests
make help       # Show all targets
```

## How It Works

The macOS Swift app:
1. Starts the Go server as a subprocess (`./momd -port 9876`)
2. Makes HTTP GET to `http://localhost:9876/` to fetch menu JSON
3. Builds a native NSMenu from the JSON structure
4. Binds each menu item to make HTTP requests to their respective paths when clicked

The Go server:
1. Defines the menu structure in code (in `main.go`)
2. Serves the menu as JSON at the root endpoint (`/`)
3. Handles menu item actions at their registered paths

## Requirements

- macOS 11.0+
- Swift (Xcode Command Line Tools)
- Go 1.21+

## Troubleshooting

### "momd binary not found" error

Run the test script to verify the app bundle:
```bash
./macos/test-app.sh
```

This will check:
- App bundle structure
- Binary location and permissions
- Server functionality

### Viewing Logs

**Option 1: Console.app** (Best for troubleshooting)
```bash
# Open Console app
open -a Console
# Search for "momd" to see all logs
```

**Option 2: Terminal log streaming** (Real-time logs)
```bash
# Stream live logs from both Swift app and Go server
log stream --predicate 'subsystem == "com.mchmarny.momd"' --level info

# View recent logs (last 5 minutes)
log show --predicate 'subsystem == "com.mchmarny.momd"' --last 5m --info

# View with debug details
log stream --predicate 'subsystem == "com.mchmarny.momd"' --level debug
```

**Option 3: Run from Terminal** (Development mode)
```bash
# Run app from terminal to see output directly
./macos/build/momd.app/Contents/MacOS/momd
```

**Option 4: Run Go server directly** (Server development only)
```bash
# Run just the Go server for testing handlers
make server
# or
./bin/momd -port 9876
```

**What You'll See:**
- Swift app logs: Server startup, menu building, user actions
- Go server logs: HTTP requests, handler execution (prefixed with `[Server]`)

**Log Format:**
```
2025-11-02 06:15:07.609652 Info momd: [com.mchmarny.momd:app] Invoking callback: /item1
2025-11-02 06:15:07.620297 Info momd: [com.mchmarny.momd:app] [Server] 2025/11/02 06:15:07 INFO handling method=GET url=/item1
```

**Note**: 
- The Swift app uses `os_log` (unified logging system) for all logs
- Go server output (stdout) is captured via pipes and forwarded to `os_log` with `[Server]` prefix
- Go server errors (stderr) are captured and logged with `[Server Error]` prefix
- All logs appear together in Console.app and `log stream` with proper timestamps

### Manual testing

Test the Go server directly:
```bash
./bin/momd -port 9999
curl http://localhost:9999/
```

## Architecture

```
┌────────────────────┐
│  macOS Menu Bar    │
│  (Swift App)       │
└────────┬───────────┘
         │ HTTP
         ↓
┌────────────────────┐
│  Go Server         │
│  (Port 9876)       │
└────────────────────┘
```

The menu package (`pkg/menu`) is reusable - you can use it in any Go application to create menu-driven interfaces!

## Disclaimer

This is my personal project and it does not represent my employer. While I do my best to ensure that everything works, I take no responsibility for issues caused by this code.