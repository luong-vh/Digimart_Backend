# Stage 1: The build stage
FROM golang:1.25.1 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /myapp

# Stage 2: The minimal runtime stage
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /myapp .
EXPOSE 8080
CMD ["./myapp"]