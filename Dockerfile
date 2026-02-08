# syntax=docker/dockerfile:1.7

FROM golang:1.25-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go install github.com/swaggo/swag/cmd/swag@v1.16.6
RUN /go/bin/swag init -g cmd/api/main.go -o docs/swagger

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/tracking-scrapper.go ./cmd/api

FROM debian:bookworm-slim AS runtime

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        chromium \
        fonts-liberation \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /out/tracking-scrapper.go /app/tracking-scrapper.go

EXPOSE 8080

ENTRYPOINT ["/app/tracking-scrapper.go"]
CMD ["-rod=bin=/usr/bin/chromium"]
