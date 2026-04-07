FROM node:24-slim@sha256:06e5c9f86bfa0aaa7163cf37a5eaa8805f16b9acb48e3f85645b09d459fc2a9f

COPY --from=ghcr.io/vertti/preflight:latest@sha256:8519b98cd654991a493514bb3dbcb0a42be82690d83633a4450c353a52bf12c8 /preflight /usr/local/bin/preflight

ENV PREFLIGHT_COLOR=1
ENV APP_ENV=production

RUN echo '{"version": "2.1.0"}' > /app.json

RUN preflight cmd node --min 18.0
RUN preflight env APP_ENV --one-of dev,staging,production
RUN preflight json /app.json --has-key version
RUN preflight cmd python3
