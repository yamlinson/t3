FROM golang:1.26.1-bookworm AS builder


WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -o t3 .

FROM debian:bookworm-slim

COPY --from=builder /app/t3 /
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

RUN adduser --disabled-password --gecos '' --shell /bin/false appuser
RUN mkdir -p /data && \
    chown appuser:appuser /data && \
    chmod 0755 /data
RUN mkdir -p /.ssh && \
    chown appuser:appuser /.ssh && \
    chmod 0755 /.ssh
USER appuser:appuser

EXPOSE 2222
CMD ["/t3"]
