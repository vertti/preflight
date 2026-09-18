# A scratch image has no CA bundle, and Go's TLS verification has nowhere else
# to look, so every `preflight http https://...` fails with "certificate signed
# by unknown authority". Alpine's bundle is ~220KB and is already a maintained
# artifact, so take it rather than shipping certificates from this repo.
FROM alpine:3.24@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6 AS certs

FROM scratch
ARG TARGETARCH
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY dist/preflight-linux-${TARGETARCH} /preflight
ENTRYPOINT ["/preflight"]
