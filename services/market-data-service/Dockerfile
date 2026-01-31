FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

COPY go.work go.work.sum ./
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY services/market-data-service/go.mod services/market-data-service/go.sum ./services/market-data-service/

RUN cd pkg && go mod download
RUN cd services/market-data-service && go mod download

COPY pkg/ ./pkg/
COPY services/market-data-service/ ./services/market-data-service/

RUN cd services/market-data-service && CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/service .

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/service .

EXPOSE 8007

CMD ["./service"]
