# Broker (relay) image — pre-built binary copied by GoReleaser
# Do not build Go here; GoReleaser injects the binary into the context.
ARG TARGETPLATFORM
FROM scratch
COPY ${TARGETPLATFORM}/relay /relay
EXPOSE 6121
ENTRYPOINT ["/relay"]
