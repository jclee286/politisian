#!/bin/bash

# Stop any currently running server process
echo "Stopping existing server..."
pkill -f politician_server || true
sleep 1

# --- Complete Initialization for a Fresh Start ---
echo "Performing complete initialization..."

# 1. Delete old blockchain data
echo "Deleting old blockchain data (.cometbft/data)..."
rm -rf .cometbft/data

# 2. Delete old application state ("memory")
echo "Deleting old application state (app_state.json)..."
rm -f app_state.json

# 3. Recreate essential folder and state file for CometBFT
echo "Creating essential CometBFT data directory..."
mkdir -p .cometbft/data
echo '{}' > .cometbft/data/priv_validator_state.json

echo "Initialization complete."
# --- End of Initialization ---

# Set Google OAuth environment variables
export GOOGLE_CLIENT_ID="152573583059-2k51btfpnqb31potv830g676nag3flps.apps.googleusercontent.com"
export GOOGLE_CLIENT_SECRET="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"

# Build the project
echo "Building the server..."
go build -o politician_server .
if [ $? -ne 0 ]; then
    echo "Build failed! Please check the errors above."
    exit 1
fi
echo "Build successful."

# Run the server in the background with log redirection
echo "Starting the server in the background..."
./politician_server > stdout.log 2> stderr.log &

# Print confirmation
echo "Server started in the background (PID: $!). Logs are in stdout.log and stderr.log."
echo "To check status after a few seconds: curl http://localhost:26657/status"
echo "To stop: pkill -f politician_server" 