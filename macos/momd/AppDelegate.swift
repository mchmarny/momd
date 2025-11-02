import Cocoa
import Foundation

class AppDelegate: NSObject, NSApplicationDelegate {
    private var statusItem: NSStatusItem!
    private var menu: NSMenu!
    private var serverProcess: Process?
    private let serverPort: Int = 9876
    private let serverPath: String
    
    override init() {
        // Determine the path to the momd binary
        // The binary is at: momd.app/Contents/MacOS/momd (executable)
        // We want: momd.app/Contents/Resources/momd (Go server)
        if let resourcePath = Bundle.main.resourcePath {
            self.serverPath = (resourcePath as NSString).appendingPathComponent("momd")
        } else {
            // Fallback to looking in the same directory as the executable
            self.serverPath = "./momd"
        }
        super.init()
        print("Server path: \(serverPath)")
    }
    
    func applicationDidFinishLaunching(_ notification: Notification) {
        // Create status bar item
        statusItem = NSStatusBar.system.statusItem(withLength: NSStatusItem.variableLength)
        
        if let button = statusItem.button {
            button.image = NSImage(systemSymbolName: "list.bullet", accessibilityDescription: "Menu")
        }
        
        // Create a temporary menu while loading
        let loadingMenu = NSMenu()
        loadingMenu.addItem(NSMenuItem(title: "Loading...", action: nil, keyEquivalent: ""))
        loadingMenu.addItem(NSMenuItem.separator())
        loadingMenu.addItem(NSMenuItem(title: "Quit", action: #selector(quit), keyEquivalent: "q"))
        statusItem.menu = loadingMenu
        
        // Start the server
        startServer()
        
        // Wait a moment for server to start, then fetch menu
        DispatchQueue.main.asyncAfter(deadline: .now() + 1.5) {
            self.fetchAndBuildMenu()
        }
    }
    
    func applicationWillTerminate(_ notification: Notification) {
        stopServer()
    }
    
    private func startServer() {
        // Check if the server binary exists
        let fileManager = FileManager.default
        if !fileManager.fileExists(atPath: serverPath) {
            showError("Server binary not found at: \(serverPath)")
            return
        }
        
        serverProcess = Process()
        serverProcess?.executableURL = URL(fileURLWithPath: serverPath)
        serverProcess?.arguments = ["-port", "\(serverPort)"]
        
        // Capture server output and forward to our logs
        let outputPipe = Pipe()
        let errorPipe = Pipe()
        serverProcess?.standardOutput = outputPipe
        serverProcess?.standardError = errorPipe
        
        // Read and log server output
        outputPipe.fileHandleForReading.readabilityHandler = { handle in
            let data = handle.availableData
            if !data.isEmpty, let output = String(data: data, encoding: .utf8) {
                print("[Server] \(output.trimmingCharacters(in: .whitespacesAndNewlines))")
            }
        }
        
        errorPipe.fileHandleForReading.readabilityHandler = { handle in
            let data = handle.availableData
            if !data.isEmpty, let output = String(data: data, encoding: .utf8) {
                print("[Server Error] \(output.trimmingCharacters(in: .whitespacesAndNewlines))")
            }
        }
        
        do {
            try serverProcess?.run()
            print("Server started on port \(serverPort)")
        } catch {
            print("Failed to start server: \(error)")
            showError("Failed to start server: \(error.localizedDescription)\nPath: \(serverPath)")
        }
    }
    
    private func stopServer() {
        serverProcess?.terminate()
        serverProcess = nil
        print("Server stopped")
    }
    
    private func fetchAndBuildMenu() {
        guard let url = URL(string: "http://localhost:\(serverPort)/") else {
            showError("Invalid server URL")
            return
        }
        
        let task = URLSession.shared.dataTask(with: url) { [weak self] data, response, error in
            guard let self = self else { return }
            
            if let error = error {
                DispatchQueue.main.async {
                    self.showError("Failed to fetch menu: \(error.localizedDescription)")
                }
                return
            }
            
            guard let data = data else {
                DispatchQueue.main.async {
                    self.showError("No data received from server")
                }
                return
            }
            
            do {
                let menuData = try JSONDecoder().decode(MenuData.self, from: data)
                DispatchQueue.main.async {
                    self.buildMenu(from: menuData)
                }
            } catch {
                DispatchQueue.main.async {
                    self.showError("Failed to parse menu: \(error.localizedDescription)")
                }
            }
        }
        task.resume()
    }
    
    private func buildMenu(from menuData: MenuData) {
        print("Building menu from data...")
        menu = NSMenu()
        
        // Add menu title if available
        if let title = menuData.title {
            let titleItem = NSMenuItem(title: title, action: nil, keyEquivalent: "")
            titleItem.isEnabled = false
            menu.addItem(titleItem)
            menu.addItem(NSMenuItem.separator())
            print("Added title: \(title)")
        }
        
        // Add menu items
        for item in menuData.items {
            addMenuItem(item, to: menu)
            print("Added menu item: \(item.title)")
        }
        
        // Add separator and quit option
        menu.addItem(NSMenuItem.separator())
        let quitItem = NSMenuItem(title: "Quit", action: #selector(quit), keyEquivalent: "q")
        quitItem.target = self
        menu.addItem(quitItem)
        
        statusItem.menu = menu
        print("Menu built successfully with \(menuData.items.count) items")
        print("Status item menu set: \(statusItem.menu != nil)")
    }
    
    private func addMenuItem(_ item: MenuItem, to menu: NSMenu) {
        let menuItem = NSMenuItem(title: item.title, action: nil, keyEquivalent: "")
        
        // Set tooltip from description if available
        if let description = item.description {
            menuItem.toolTip = description
        }
        
        // If item has children, create submenu
        if let children = item.items, !children.isEmpty {
            let submenu = NSMenu(title: item.title)
            for child in children {
                addMenuItem(child, to: submenu)
            }
            menuItem.submenu = submenu
        } else if let path = item.path {
            // If item has a path (handler), make it actionable
            menuItem.target = self
            menuItem.action = #selector(handleMenuItemAction(_:))
            menuItem.representedObject = path
        }
        
        menu.addItem(menuItem)
    }
    
    @objc private func handleMenuItemAction(_ sender: NSMenuItem) {
        guard let path = sender.representedObject as? String else { return }
        
        guard let url = URL(string: "http://localhost:\(serverPort)\(path)") else {
            showError("Invalid URL for path: \(path)")
            return
        }
        
        print("Invoking menu item action: \(path)")
        
        let task = URLSession.shared.dataTask(with: url) { data, response, error in
            if let error = error {
                DispatchQueue.main.async {
                    self.showError("Failed to invoke action: \(error.localizedDescription)")
                }
                return
            }
            
            if let data = data, let responseString = String(data: data, encoding: .utf8) {
                print("Response from \(path): \(responseString)")
                // Optionally show a notification or update UI
            }
        }
        task.resume()
    }
    
    @objc private func quit() {
        // Stop the server before terminating the app
        stopServer()
        NSApplication.shared.terminate(nil)
    }
    
    private func showError(_ message: String) {
        let alert = NSAlert()
        alert.messageText = "Error"
        alert.informativeText = message
        alert.alertStyle = .warning
        alert.addButton(withTitle: "OK")
        alert.runModal()
    }
}

// MARK: - Data Models

struct MenuData: Codable {
    let title: String?
    let description: String?
    let items: [MenuItem]
}

struct MenuItem: Codable {
    let type: String
    let path: String?
    let title: String
    let description: String?
    let items: [MenuItem]?
}

struct MenuItemAction {
    let type: String
    let path: String
}
