# syntax=docker/dockerfile:1
FROM golang:1.24-alpine AS builder
WORKDIR /build
COPY go.mod ./
COPY main.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /workflow-hello-weather .

FROM alpine:3.21
COPY --from=builder /workflow-hello-weather /usr/local/bin/workflow-hello-weather
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/workflow-hello-weather"]
