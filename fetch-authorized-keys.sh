#!/bin/bash
# filepath: /usr/local/bin/fetch-ssh-keys.sh

# Configuration
CACHE_DIR="/var/cache/ssh-keys"
CACHE_FILE="${CACHE_DIR}/$(hostname).keys"
SERVER_URL="http://localhost:8080/keys/$(hostname)"
AUTH_TOKEN="secret-token-1"

# Create cache directory if it doesn't exist
mkdir -p "$CACHE_DIR"

# Main logic
# Try to fetch new keys
new_keys=$(curl -s -f -H "Authorization: Token $AUTH_TOKEN" "$SERVER_URL")
if [ $? -eq 0 ] && [ ! -z "$new_keys" ]; then
    echo "$new_keys" > "$CACHE_FILE"
    echo "$new_keys"
elif [ -f "$CACHE_FILE" ]; then
    # Server unreachable, use cache if it exists
    cat "$CACHE_FILE"
else
    # No cache and server unreachable
    echo "" # No keys
fi
