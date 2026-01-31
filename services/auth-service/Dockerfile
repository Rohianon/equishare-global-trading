FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

COPY go.work go.work.sum ./
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY services/auth-service/go.mod services/auth-service/go.sum ./services/auth-service/

RUN cd pkg && go mod download
RUN cd services/auth-service && go mod download

COPY pkg/ ./pkg/
COPY services/auth-service/ ./services/auth-service/

RUN cd services/auth-service && CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/service .

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/service .

EXPOSE 8001

CMD ["./service"]
