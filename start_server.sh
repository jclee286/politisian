#!/bin/bash

# Stop any currently running server process
echo "Stopping existing server..."
pkill -f politician_server || true
sleep 1

# --- Complete Initialization for a Fresh Start ---
echo "Performing complete initialization..."

# 1. Delete old blockchain data and application state
echo "Deleting old blockchain data and application state..."
rm -rf .cometbft
rm -f app_state.json

echo "Initialization complete."
# --- End of Initialization ---

# Set Google OAuth environment variables
export GOOGLE_CLIENT_ID="152573583059-2k51btfpnqb31potv830g676nag3flps.apps.googleusercontent.com"
export GOOGLE_CLIENT_SECRET="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"

# Build the project
echo "Building the server..."
mkdir -p ./build_temp
TMPDIR=./build_temp go build -o politician_server .
if [ $? -ne 0 ]; then
    echo "Build failed! Please check the errors above."
    exit 1
fi
echo "Build successful."

# Initialize CometBFT using the newly built binary
# This creates all necessary files (genesis.json, config.toml, node_key.json, priv_validator_key.json)
echo "Initializing CometBFT..."
./politician_server init
if [ $? -ne 0 ]; then
    echo "CometBFT initialization failed!"
    exit 1
fi
echo "CometBFT initialized."

# Run the server in the background with log redirection
echo "Starting the server in the background..."
./politician_server > stdout.log 2> stderr.log &

# Print confirmation
echo "Server started in the background (PID: $!). Logs are in stdout.log and stderr.log."
echo "To check status after a few seconds: curl http://localhost:26657/status" 