FROM golang:1.22.3 as builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN go build -trimpath ./cmd/plc-mirror

FROM alpine:latest as certs
RUN apk --update add ca-certificates curl

FROM debian:stable-slim
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /app/plc-mirror .
ENTRYPOINT ["./plc-mirror"]
