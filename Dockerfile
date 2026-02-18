# Broker (relay) image — pre-built binary copied by GoReleaser
# Do not build Go here; GoReleaser injects the binary into the context.
FROM scratch
ARG TARGETPLATFORM
COPY ${TARGETPLATFORM}/relay /relay
EXPOSE 6121
ENTRYPOINT ["/relay"]
