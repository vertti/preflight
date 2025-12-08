# Example: Runtime checks with entrypoint script
#
# Preflight checks run at container startup before your app starts.
# The container waits for dependencies (postgres) and validates its
# environment before starting the main process.
#
# Build: docker build -t myapp .
# Run:   docker run -e DATABASE_URL=postgres://... myapp

FROM alpine:3.19

COPY --from=ghcr.io/vertti/preflight:latest /preflight /usr/local/bin/preflight

WORKDIR /app
COPY entrypoint.sh /entrypoint.sh
COPY myapp /usr/local/bin/myapp
COPY config.yaml /app/config.yaml

RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
CMD ["myapp"]
