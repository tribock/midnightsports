# Frontend bauen
FROM node:22-alpine AS web
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# Backend bauen
FROM golang:1.26-alpine AS api
WORKDIR /src
COPY api/go.mod api/go.sum ./
RUN go mod download
COPY api/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /server ./cmd/server

# Laufzeit-Image (enthält CA-Zertifikate für Google OAuth)
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=api /server /app/server
COPY --from=web /web/dist /app/web
ENV STATIC_DIR=/app/web \
    PORT=8080
EXPOSE 8080
ENTRYPOINT ["/app/server"]
