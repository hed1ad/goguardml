# Build stage
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git libpcap-dev gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o goanomaly ./cmd/goanomaly

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache libpcap ca-certificates

WORKDIR /app

COPY --from=builder /app/goanomaly .

RUN adduser -D -g '' appuser
USER appuser

ENTRYPOINT ["./goanomaly"]
CMD ["--help"]
