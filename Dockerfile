FROM scratch
ARG TARGETARCH
COPY dist/preflight-linux-${TARGETARCH} /preflight
ENTRYPOINT ["/preflight"]
