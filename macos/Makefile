.PHONY: app clean-app run

app: build ## Build the macOS menu bar app
	@echo "Building macOS app..."
	@mkdir -p macos/build/momd.app/Contents/MacOS
	@mkdir -p macos/build/momd.app/Contents/Resources
	
	# Compile Swift code
	swiftc -o macos/build/momd.app/Contents/MacOS/momd \
		-framework Cocoa \
		-framework Foundation \
		macos/momd/AppDelegate.swift
	
	# Copy Info.plist
	cp macos/momd/Info.plist macos/build/momd.app/Contents/Info.plist
	
	# Copy the Go binary to Resources
	cp bin/momd macos/build/momd.app/Contents/Resources/momd
	
	@echo "macOS app built at: macos/build/momd.app"
	@echo "To run: open macos/build/momd.app"

clean-app: ## Clean the macOS build artifacts
	rm -rf macos/build

run: app ## Build and run the macOS app
	open macos/build/momd.app
