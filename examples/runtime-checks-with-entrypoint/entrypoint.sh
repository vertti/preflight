#!/bin/sh
# Example: Runtime checks before app starts
#
# This entrypoint script runs preflight checks at container startup,
# waiting for dependencies and validating the environment before
# starting the main process.

set -e

preflight tcp postgres:5432 --timeout 30s
preflight env DATABASE_URL
preflight file /app/config.yaml --not-empty

exec "$@"
