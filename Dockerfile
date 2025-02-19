FROM golang:1.21-alpine AS builder
MAINTAINER <Muutassim Mukhtar>
WORKDIR /webhook
COPY . . 
RUN go mod tidy && GOOS=linux GOARCH=amd64 go build -o webhook-server ./webhook

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /webhook/webhook-server .
EXPOSE 8443
CMD ["./webhook-server"]