FROM golang:1.25-alpine AS builder

ARG VERSION=0.2.4
ARG COMMIT=unknown
ARG BUILD_TIME=unknown

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags="-s -w -X github.com/topbase/topbase/internal/buildinfo.Version=${VERSION} -X github.com/topbase/topbase/internal/buildinfo.Commit=${COMMIT} -X github.com/topbase/topbase/internal/buildinfo.BuildTime=${BUILD_TIME}" \
    -o /out/topbase ./cmd/topbase \
    && CGO_ENABLED=0 go build -trimpath \
    -ldflags="-s -w -X github.com/topbase/topbase/internal/buildinfo.Version=${VERSION} -X github.com/topbase/topbase/internal/buildinfo.Commit=${COMMIT} -X github.com/topbase/topbase/internal/buildinfo.BuildTime=${BUILD_TIME}" \
    -o /out/topbase-backup ./cmd/topbase-backup \
    && CGO_ENABLED=0 go build -trimpath \
    -ldflags="-s -w -X github.com/topbase/topbase/internal/buildinfo.Version=${VERSION} -X github.com/topbase/topbase/internal/buildinfo.Commit=${COMMIT} -X github.com/topbase/topbase/internal/buildinfo.BuildTime=${BUILD_TIME}" \
    -o /out/topbase-cli ./cmd/topbase-cli \
    && CGO_ENABLED=0 go build -trimpath \
    -ldflags="-s -w -X github.com/topbase/topbase/internal/buildinfo.Version=${VERSION} -X github.com/topbase/topbase/internal/buildinfo.Commit=${COMMIT} -X github.com/topbase/topbase/internal/buildinfo.BuildTime=${BUILD_TIME}" \
    -o /out/topbase-mcp ./cmd/topbase-mcp

FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S -g 10001 topbase \
    && adduser -S -D -H -u 10001 -G topbase topbase \
    && mkdir -p /data /backups \
    && chown topbase:topbase /data /backups

WORKDIR /app
COPY --from=builder /out/topbase /app/topbase
COPY --from=builder /out/topbase-backup /app/topbase-backup
COPY --from=builder /out/topbase-cli /app/topbase-cli
COPY --from=builder /out/topbase-mcp /app/topbase-mcp

USER topbase
ENV TOPBASE_ADDR=:8080
ENV TOPBASE_DATA_DIR=/data
EXPOSE 8080
VOLUME ["/data", "/backups"]
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 CMD wget -qO- http://127.0.0.1:8080/api/ready || exit 1

ENTRYPOINT ["/app/topbase"]
