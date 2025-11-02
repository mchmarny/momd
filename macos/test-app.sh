#!/bin/bash

echo "Testing macOS app bundle..."

APP_PATH="macos/build/momd.app"

echo "1. Checking app structure..."
ls -la "$APP_PATH/Contents/"

echo -e "\n2. Checking Resources..."
ls -la "$APP_PATH/Contents/Resources/"

echo -e "\n3. Checking if momd binary is executable..."
file "$APP_PATH/Contents/Resources/momd"

echo -e "\n4. Testing momd binary directly..."
"$APP_PATH/Contents/Resources/momd" -port 9999 &
PID=$!
sleep 2

echo -e "\n5. Testing server endpoint..."
curl -s http://localhost:9999/ | jq . || echo "Server not responding"

echo -e "\n6. Killing test server..."
kill $PID 2>/dev/null

echo -e "\nTest complete! If all steps passed, the app should work."
echo "Run with: make run"
