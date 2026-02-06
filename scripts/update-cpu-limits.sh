#!/bin/bash

# Usage message
if [ -z "$1" ]; then
  echo "Usage: $0 <cpu_limit> <docker_compose_file>"
  echo "Example: $0 0.5"
  echo "Example with compose file: $0 1.0 docker-compose.prod.yml"
  exit 1
fi

# Get container IDs managed by Compose (use the compose file if provided in arguments)
if [ -n "$2" ]; then
  COMPOSE_FILE="$2"
  CONTAINERS=$(docker compose -f "$COMPOSE_FILE" ps -q)
else
  CONTAINERS=$(docker compose ps -q)
fi

CPULIMIT="$1"

# Update CPU limit for each container
for CONTAINER in $CONTAINERS; do
  echo "Setting CPU limit to $CPULIMIT for container $CONTAINER"
  docker update --cpus="$CPULIMIT" "$CONTAINER"
done

echo "All containers updated with CPU limit of $CPULIMIT"