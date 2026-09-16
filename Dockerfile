# syntax=docker/dockerfile:1.7

FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build

ARG TARGETOS
ARG TARGETARCH

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN mkdir -p /out/data \
    && CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
        -trimpath -ldflags="-s -w" -o /out/platform ./cmd/platform

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/platform /app/platform
COPY --from=build --chown=65532:65532 /src/web/static /app/web/static
COPY --from=build --chown=65532:65532 /out/data /var/lib/tockrplatform

WORKDIR /app
USER 65532:65532
ENV PLATFORM_DB_PATH=/var/lib/tockrplatform/platform.db
EXPOSE 8080
VOLUME ["/var/lib/tockrplatform"]
ENTRYPOINT ["/app/platform"]
