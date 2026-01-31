# Use distroless as minimal base image
FROM gcr.io/distroless/static:nonroot
ARG TARGETOS=linux
ARG TARGETARCH=amd64

WORKDIR /
COPY bin/manager-${TARGETOS}-${TARGETARCH} /manager
USER 65532:65532

ENTRYPOINT ["/manager"]
