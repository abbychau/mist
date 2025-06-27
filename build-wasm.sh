#!/bin/bash

# Define the output directory for WASM files
OUTPUT_DIR="./docs"

# Ensure the output directory exists
mkdir -p "$OUTPUT_DIR"

# Define the output file paths
OUTPUT_WASM="$OUTPUT_DIR/mist.wasm"
OUTPUT_JS="$OUTPUT_DIR/wasm_exec.js"

# Build the WASM module
echo "Building Mist WASM..."
env GOOS=js GOARCH=wasm go build -o "$OUTPUT_WASM" ./wasm

# Check if the build was successful
if [ $? -eq 0 ]; then
  echo "✅ WASM build successful!"
  
  # Copy wasm_exec.js to the output directory
  cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" "$OUTPUT_JS"
  echo "✅ wasm_exec.js copied to $OUTPUT_DIR"
else
  echo "❌ WASM build failed!"
fi