FROM node:24-slim@sha256:55fb5f5a527ea2c5c4a95f963e48ef90aa38c7b346054fdc8ee9957a2b55d205

COPY --from=ghcr.io/vertti/preflight:latest@sha256:637e113c1c353048991fd0acbb00e834c9d457bda4dc223a3f26e5b5d43f4667 /preflight /usr/local/bin/preflight

ENV PREFLIGHT_COLOR=1
ENV APP_ENV=production

RUN echo '{"version": "2.1.0"}' > /app.json

RUN preflight cmd node --min 18.0
RUN preflight env APP_ENV --one-of dev,staging,production
RUN preflight json /app.json --has-key version
RUN preflight cmd python3
