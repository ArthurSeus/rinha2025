# Etapa 1: build
FROM golang:1.24.5-alpine AS builder

WORKDIR /app
COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . ./
RUN go build -o api ./cmd/main.go

# Etapa 2: imagem final
FROM alpine:3.20

WORKDIR /app

COPY --from=builder /app/api .
#COPY --from=builder /app/config.yaml .

EXPOSE 8000

CMD ["./api"]