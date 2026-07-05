FROM golang:1.25-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY go/lib ./go/lib
COPY go/pkg ./go/pkg
COPY demo/grpc ./demo/grpc
RUN CGO_ENABLED=0 go build -tags debugviz -o /demo-grpc ./demo/grpc

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=builder /demo-grpc /usr/local/bin/demo-grpc
EXPOSE 9090
CMD ["demo-grpc"]
