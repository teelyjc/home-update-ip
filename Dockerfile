FROM golang:1.24 AS builder
WORKDIR /app

COPY . .

RUN go mod download

RUN CGO_ENABLED=0 \
  GOOS=linux \
  go build -o /dist/app ./cmd/app/main.go

RUN ls -la /dist/

FROM alpine AS runner
WORKDIR /app

COPY --from=builder /dist/ ./dist/

ENV ENVIRONMENT=production

CMD ["./dist/app"]
