FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

COPY go.work go.work.sum ./
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY services/ussd-service/go.mod services/ussd-service/go.sum ./services/ussd-service/

RUN cd pkg && go mod download
RUN cd services/ussd-service && go mod download

COPY pkg/ ./pkg/
COPY services/ussd-service/ ./services/ussd-service/

RUN cd services/ussd-service && CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/service .

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/service .

EXPOSE 8005

CMD ["./service"]
