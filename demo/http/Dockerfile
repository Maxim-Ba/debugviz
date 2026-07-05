FROM golang:1.25-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY go/lib ./go/lib
COPY go/pkg ./go/pkg
COPY go/cmd ./go/cmd
COPY debugviz.yaml ./
COPY demo/http ./demo/http
RUN go run ./go/cmd/debugviz wire --config debugviz.yaml --write ./demo/http
RUN CGO_ENABLED=0 go build -tags debugviz -o /demo-http ./demo/http

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=builder /demo-http /usr/local/bin/demo-http
EXPOSE 8080
CMD ["demo-http"]
