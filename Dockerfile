# A scratch image has no CA bundle, and Go's TLS verification has nowhere else
# to look, so every `preflight http https://...` fails with "certificate signed
# by unknown authority". Alpine's bundle is ~220KB and is already a maintained
# artifact, so take it rather than shipping certificates from this repo.
FROM alpine:3.24@sha256:c6b073f8e9e58d881f6ffba6ebf3b9ebd60b9e7d6e8337018c0e68bb5c356873 AS certs

FROM scratch
ARG TARGETARCH
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY dist/preflight-linux-${TARGETARCH} /preflight
ENTRYPOINT ["/preflight"]
