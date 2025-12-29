FROM golang:1.24-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

WORKDIR /app/orders/cmd
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o orders .

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/
COPY --from=builder /app/orders/cmd .
COPY --from=builder /app/orders/migrations ./migrations

EXPOSE 8082
ENV PORT=8082

CMD ["./orders"]