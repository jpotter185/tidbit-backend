#!/bin/bash

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${GREEN}Starting tidbit deployment...${NC}"

# Stop existing containers
echo -e "${YELLOW}Stopping existing containers...${NC}"
docker compose down

# Start containers with rebuild
echo -e "${YELLOW}Starting Docker containers...${NC}"
docker compose up --build -d

if [ $? -ne 0 ]; then
    echo -e "${RED}Failed to start Docker containers!${NC}"
    exit 1
fi

sleep 3

echo -e "${GREEN}Containers started successfully!${NC}"
docker compose ps

echo -e "${YELLOW}Showing logs (Ctrl+C to exit)...${NC}"
docker compose logs -f