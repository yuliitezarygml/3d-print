FROM golang:1.25-alpine AS build
WORKDIR /src
COPY apps/backend/go.mod apps/backend/go.sum* ./
RUN go mod download
COPY apps/backend/ .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata && addgroup -S app && adduser -S app -G app
WORKDIR /app
COPY --from=build /out/api /app/api
RUN mkdir -p /data/uploads && chown -R app:app /data
USER app
EXPOSE 8080
ENTRYPOINT ["/app/api"]
