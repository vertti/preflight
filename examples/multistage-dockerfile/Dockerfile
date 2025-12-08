# Example: Multi-stage build with preflight validation
#
# The validate stage runs checks during build but doesn't ship preflight
# in the final image. If any check fails, the build fails.
#
# Build: docker build -t myapp .

# Stage 1: Build
FROM golang:1.21 AS build
WORKDIR /app
COPY . .
RUN go build -o myapp

# Stage 2: Validate (runs checks, not included in final image)
FROM alpine:3.19 AS validate
COPY --from=ghcr.io/vertti/preflight:latest /preflight /usr/local/bin/preflight
COPY --from=build /app/myapp /usr/local/bin/myapp
RUN preflight cmd myapp --min 2.0
RUN preflight file /usr/local/bin/myapp --executable

# Stage 3: Final (clean, no preflight)
FROM alpine:3.19
COPY --from=build /app/myapp /usr/local/bin/myapp
CMD ["myapp"]
