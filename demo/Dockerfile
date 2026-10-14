FROM node:24-slim@sha256:a5f4366e834695d5f363b920c104953a535d3a9864d1bdf74566a0ebe01daee5

COPY --from=ghcr.io/vertti/preflight:latest@sha256:637e113c1c353048991fd0acbb00e834c9d457bda4dc223a3f26e5b5d43f4667 /preflight /usr/local/bin/preflight

ENV PREFLIGHT_COLOR=1
ENV APP_ENV=production

RUN echo '{"version": "2.1.0"}' > /app.json

RUN preflight cmd node --min 18.0
RUN preflight env APP_ENV --one-of dev,staging,production
RUN preflight json /app.json --has-key version
RUN preflight cmd python3
