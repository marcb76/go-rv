#!/bin/bash

###################################################################################
# Script to set up the Gemini API key, build, and run the go-rv application
# Usage: ./run.sh [--noRun]

# Defines
KEY_FILE="assets/gemini/gemini_api_key.txt"




# 0. Exit immediately if a command exits with a non-zero status
set -e

# 1. Parse arguments for --noRun flag (optional)
NORUN=false
for arg in "$@"; do
    if [ "$arg" == "--noRun" ]; then
        NORUN=true
        break
    fi
done

# 2. Check if the API key file exists locally
echo "Setting up gemini API key..."
if [ ! -f "$KEY_FILE" ]; then
    echo "Error: API key file not found at $KEY_FILE" >&2
    exit 1
fi

# 3. Read and export the GEMINI_API_KEY from the file (trimming potential whitespace or newlines)
export GEMINI_API_KEY=$(cat "$KEY_FILE" | tr -d '\r' | xargs)
if [ -z "$GEMINI_API_KEY" ]; then
    echo "Error: The API key file is empty." >&2
    exit 1
fi

# 4. Compile the application
echo "Building go-rv..."
go build -o go-rv.exe .

# 5. Execute the application (conditioned on --noRun)
if [ "$NORUN" = true ]; then
    echo "Build complete successfully. Skipping execution (--noRun specified)."
else
    echo "Starting go-rv service..."
    ./go-rv.exe
fi