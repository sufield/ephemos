#!/bin/bash

echo "Cleaning up ALL echo-server processes and releasing ports..."

# Kill all user-owned echo-server processes
echo "Killing user-owned echo-server processes..."
pkill -u $(whoami) -f echo-server 2>/dev/null || true

# Wait a moment for processes to die
sleep 2

# Force kill any stubborn processes using demo ports
echo "Force killing processes on demo ports..."
for port in {50051..50063}; do
    PIDS=$(lsof -ti :$port 2>/dev/null | grep -v "^$" || true)
    if [ -n "$PIDS" ]; then
        echo "Killing processes using port $port: $PIDS"
        echo "$PIDS" | xargs -r kill -9 2>/dev/null || true
    fi
done

# Final verification
sleep 1
REMAINING_PORTS=$(ss -tulpn | grep -E ":(5005[0-9]|5006[0-9])" || true)
if [ -n "$REMAINING_PORTS" ]; then
    echo "⚠️  Some ports still in use:"
    echo "$REMAINING_PORTS"
else
    echo "✅ All demo ports are now clean!"
fi