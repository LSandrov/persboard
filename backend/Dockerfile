FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/persboard-api ./cmd/server

FROM alpine:3.20

RUN adduser -D -H appuser
USER appuser

WORKDIR /srv
COPY --from=builder /bin/persboard-api /srv/persboard-api

EXPOSE 8080

CMD ["/srv/persboard-api"]
