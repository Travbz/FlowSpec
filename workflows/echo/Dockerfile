# syntax=docker/dockerfile:1
FROM golang:1.24-alpine AS builder
WORKDIR /build
COPY go.mod ./
COPY main.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /workflow-echo .

FROM alpine:3.21
COPY --from=builder /workflow-echo /usr/local/bin/workflow-echo
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/workflow-echo"]
