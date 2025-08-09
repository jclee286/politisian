#!/bin/bash

# Stop any currently running server process
echo "Stopping existing server..."
pkill -f politician_server || true
sleep 1

# --- Complete Initialization for a Fresh Start ---
echo "Performing complete initialization..."

# 1. Delete old blockchain data
echo "Deleting old blockchain data..."
rm -rf .cometbft
rm -f app_state.json

# 2. Initialize CometBFT using the official command
# This creates all necessary files (genesis.json, config.toml, node_key.json, priv_validator_key.json)
echo "Initializing CometBFT..."
cometbft init --home .cometbft
if [ $? -ne 0 ]; then
    echo "CometBFT initialization failed!"
    exit 1
fi
echo "CometBFT initialized."

# Set environment variables (Privy keys, etc.)
export PRIVY_APP_ID="cme44fxu403c5lb0b0dv9lq31"
export PRIVY_APP_SECRET="4WYnujvc9uzSjPWG816i6N6c9ay3qTrULRRL6jAfzWtLfRk2WypE1jopHB2sCjhyvSYW5hzhqZW6nXyGMy31VsLQ"

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