FROM golang:1.23-alpine AS builder

RUN apk --no-cache add ca-certificates git

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o t3 .

FROM gcr.io/distroless/static:nonroot

COPY --from=builder --chown=nonroot:nonroot /app/t3 /

USER nonroot:nonroot

EXPOSE 2222

CMD ["/t3"]
