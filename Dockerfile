FROM golang:alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o app

FROM scratch
COPY --from=builder /app/app /app/app
ENTRYPOINT ["/app/app"]