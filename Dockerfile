FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod ./
COPY *.go ./
RUN CGO_ENABLED=0 go build -o /service .

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /service .
EXPOSE 8080
CMD ["./service"]
