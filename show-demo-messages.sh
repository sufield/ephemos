#!/bin/bash

# Colors
GREEN='\033[32m'
RESET='\033[0m'

echo -e "\n${GREEN}╔══════════════════════════════════════════════════════════════════════════════╗"
echo -e "║                               ECHO SERVER MESSAGES                          ║"
echo -e "╚══════════════════════════════════════════════════════════════════════════════╝${RESET}"

while IFS= read -r line; do
    echo -e "${GREEN}║${RESET} $line"
done < server-demo.log

echo -e "${GREEN}╚══════════════════════════════════════════════════════════════════════════════╝${RESET}"

echo -e "\n${GREEN}╔══════════════════════════════════════════════════════════════════════════════╗"
echo -e "║                               ECHO CLIENT MESSAGES                          ║"
echo -e "╚══════════════════════════════════════════════════════════════════════════════╝${RESET}"

# Run client and capture output
EPHEMOS_CONFIG=config/echo-client.yaml ./bin/echo-client 2>&1 | while IFS= read -r line; do
    echo -e "${GREEN}║${RESET} $line"
done

echo -e "${GREEN}╚══════════════════════════════════════════════════════════════════════════════╝${RESET}\n"