FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/worker ./cmd/worker

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata wget
WORKDIR /app
COPY --from=builder /bin/worker .
EXPOSE 8082
CMD ["./worker"]
