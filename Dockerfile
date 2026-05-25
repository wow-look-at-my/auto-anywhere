FROM scratch

ARG VERSION=dev

LABEL org.opencontainers.image.source="https://github.com/wow-look-at-my/auto-anywhere"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.description="Anthropic API reverse proxy that forces thinking summaries and enables auto mode"

COPY --from=gcr.io/distroless/static-debian12 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --chmod=755 build/auto-anywhere_linux_amd64 /auto-anywhere

EXPOSE 18080
STOPSIGNAL SIGTERM

ENTRYPOINT ["/auto-anywhere"]
