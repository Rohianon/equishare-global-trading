FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

COPY go.work go.work.sum ./
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY services/payment-service/go.mod services/payment-service/go.sum ./services/payment-service/

RUN cd pkg && go mod download
RUN cd services/payment-service && go mod download

COPY pkg/ ./pkg/
COPY services/payment-service/ ./services/payment-service/

RUN cd services/payment-service && CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/service .

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/service .

EXPOSE 8004

CMD ["./service"]
