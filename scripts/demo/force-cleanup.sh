#!/bin/bash
set -e

echo "Performing forceful cleanup of all SPIRE and demo processes..."

# Kill processes by port (using lsof/fuser if available)
echo "Killing processes using ports 8081, 50051..."
sudo lsof -ti :8081 | xargs -r sudo kill -9 2>/dev/null || true
sudo lsof -ti :50051 | xargs -r sudo kill -9 2>/dev/null || true
sudo lsof -ti :50052 | xargs -r sudo kill -9 2>/dev/null || true
sudo lsof -ti :50055 | xargs -r sudo kill -9 2>/dev/null || true
sudo lsof -ti :50056 | xargs -r sudo kill -9 2>/dev/null || true

# Kill processes by name
echo "Killing all SPIRE and echo processes..."
sudo pkill -9 -f spire-server 2>/dev/null || true
sudo pkill -9 -f spire-agent 2>/dev/null || true
sudo pkill -9 -f echo-server 2>/dev/null || true
pkill -9 -f echo-server 2>/dev/null || true
pkill -9 -f echo-client 2>/dev/null || true

# Clean up sockets
echo "Cleaning up sockets..."
sudo rm -rf /tmp/spire-server 2>/dev/null || true
sudo rm -rf /tmp/spire-agent 2>/dev/null || true

# Clean up log files
echo "Cleaning up log files..."
rm -f *.log 2>/dev/null || true
rm -f /tmp/server*.log 2>/dev/null || true

# Stop systemd services if they exist
echo "Stopping systemd services..."
sudo systemctl stop spire-server 2>/dev/null || true
sudo systemctl stop spire-agent 2>/dev/null || true

echo "✓ Forceful cleanup completed!"