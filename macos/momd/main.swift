import Cocoa

let app = NSApplication.shared
// Don't set activation policy - LSUIElement in Info.plist handles this

let delegate = AppDelegate()
app.delegate = delegate

app.run()
