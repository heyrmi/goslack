# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o main main.go

# Run stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates wget
WORKDIR /app

# Download wait-for script
RUN wget -qO /app/wait-for.sh https://raw.githubusercontent.com/eficode/wait-for/v2.2.3/wait-for \
    && chmod +x /app/wait-for.sh

COPY --from=builder /app/main .
COPY app.env .

EXPOSE 8080
CMD ["/app/main"]
