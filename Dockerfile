# A scratch image has no CA bundle, and Go's TLS verification has nowhere else
# to look, so every `preflight http https://...` fails with "certificate signed
# by unknown authority". Alpine's bundle is ~220KB and is already a maintained
# artifact, so take it rather than shipping certificates from this repo.
FROM alpine:3.21@sha256:48b0309ca019d89d40f670aa1bc06e426dc0931948452e8491e3d65087abc07d AS certs

FROM scratch
ARG TARGETARCH
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY dist/preflight-linux-${TARGETARCH} /preflight
ENTRYPOINT ["/preflight"]
