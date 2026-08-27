FROM golang:1.25-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/topbase ./cmd/topbase

FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S topbase \
    && adduser -S -G topbase topbase \
    && mkdir -p /data \
    && chown topbase:topbase /data

WORKDIR /app
COPY --from=builder /out/topbase /app/topbase

USER topbase
ENV TOPBASE_ADDR=:8080
ENV TOPBASE_DATA_DIR=/data
EXPOSE 8080
VOLUME ["/data"]
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 CMD wget -qO- http://127.0.0.1:8080/api/health || exit 1

ENTRYPOINT ["/app/topbase"]
