FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

COPY go.work go.work.sum ./
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY services/portfolio-service/go.mod services/portfolio-service/go.sum ./services/portfolio-service/

RUN cd pkg && go mod download
RUN cd services/portfolio-service && go mod download

COPY pkg/ ./pkg/
COPY services/portfolio-service/ ./services/portfolio-service/

RUN cd services/portfolio-service && CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/service .

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/service .

EXPOSE 8008

CMD ["./service"]
