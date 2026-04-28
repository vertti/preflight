FROM node:24-slim@sha256:03eae3ef7e88a9de535496fb488d67e02b9d96a063a8967bae657744ecd513f2

COPY --from=ghcr.io/vertti/preflight:latest@sha256:8519b98cd654991a493514bb3dbcb0a42be82690d83633a4450c353a52bf12c8 /preflight /usr/local/bin/preflight

ENV PREFLIGHT_COLOR=1
ENV APP_ENV=production

RUN echo '{"version": "2.1.0"}' > /app.json

RUN preflight cmd node --min 18.0
RUN preflight env APP_ENV --one-of dev,staging,production
RUN preflight json /app.json --has-key version
RUN preflight cmd python3
