#!/bin/bash

echo "Cleaning up Ephemos demo..."

# Stop any running processes
echo "Stopping running processes..."
pkill -f echo-server 2>/dev/null || true
pkill -f echo-client 2>/dev/null || true

# Stop SPIRE services
echo "Stopping SPIRE services..."
sudo systemctl stop spire-agent 2>/dev/null || true
sudo systemctl stop spire-server 2>/dev/null || true

# Disable services
echo "Disabling SPIRE services..."
sudo systemctl disable spire-agent 2>/dev/null || true
sudo systemctl disable spire-server 2>/dev/null || true

# Remove service files
echo "Removing service files..."
sudo rm -f /etc/systemd/system/spire-server.service
sudo rm -f /etc/systemd/system/spire-agent.service

# Clean up data
echo "Cleaning up SPIRE data..."
sudo rm -rf /opt/spire/data/*
sudo rm -rf /tmp/spire-server
sudo rm -rf /tmp/spire-agent

# Remove binaries (optional)
read -p "Remove SPIRE installation? (y/N) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "Removing SPIRE installation..."
    sudo rm -rf /opt/spire
    sudo rm -f /usr/local/bin/spire-server
    sudo rm -f /usr/local/bin/spire-agent
fi

# Reload systemd
sudo systemctl daemon-reload

echo "Cleanup completed!"