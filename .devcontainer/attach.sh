#!/bin/bash

# Attach to the running drift devcontainer.
# Exits with an error if the container is not currently running.

set -euo pipefail

# Container is identified by the `drift.role` label set in docker-compose.yml.
CONTAINER_LABEL="drift.role=devcontainer"

CONTAINER_NAME="$(docker ps -q -f "label=${CONTAINER_LABEL}")"

if [ -z "${CONTAINER_NAME}" ]; then
    echo "No running container with label '${CONTAINER_LABEL}'." >&2
    exit 1
fi

exec docker exec -it --user vscode:vscode -w /workspaces/drift "${CONTAINER_NAME}" /bin/bash
