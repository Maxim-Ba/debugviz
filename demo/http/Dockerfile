FROM golang:1.23-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY demo/http ./demo/http
RUN CGO_ENABLED=0 go build -o /demo-http ./demo/http

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=builder /demo-http /usr/local/bin/demo-http
EXPOSE 8080
CMD ["demo-http"]
