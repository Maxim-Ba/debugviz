FROM golang:1.25-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY go/lib ./go/lib
COPY go/pkg ./go/pkg
COPY go/cmd ./go/cmd
COPY demo/worker ./demo/worker
RUN go run ./go/cmd/debugviz wire --config demo/worker/debugviz.yaml --write ./demo/worker
RUN CGO_ENABLED=0 go build -tags debugviz -o /demo-worker ./demo/worker

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=builder /demo-worker /usr/local/bin/demo-worker
CMD ["demo-worker"]
