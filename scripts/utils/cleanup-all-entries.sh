#!/bin/bash

echo "Cleaning up ALL SPIRE entries for demo..."

# Get all entry IDs and delete them
sudo spire-server entry show -socketPath /tmp/spire-server/private/api.sock | grep "Entry ID" | awk '{print $4}' | while read entry_id; do
    if [ "$entry_id" != "(none)" ] && [ -n "$entry_id" ]; then
        echo "Deleting entry: $entry_id"
        sudo spire-server entry delete -socketPath /tmp/spire-server/private/api.sock -entryID "$entry_id" 2>/dev/null || true
    fi
done

echo "Cleanup complete."
echo "Current entries:"
sudo spire-server entry show -socketPath /tmp/spire-server/private/api.sock