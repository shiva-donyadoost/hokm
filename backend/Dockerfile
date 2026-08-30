# Production image: backend binary + prebuilt frontend baked in.
# Build context is the repository root (compose sets context: .).
# Stages:
#   web   - builds the frontend (only needed for the prod target)
#   build - compiles the Go backend
#   base  - minimal runtime WITHOUT embedded frontend (dev target)
#   prod  - base + baked frontend, served via WEB_DIR

FROM node:24-alpine AS web
WORKDIR /app
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY frontend/ .
RUN npm run build

FROM golang:1.27-alpine AS build
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ .
RUN CGO_ENABLED=0 go build -trimpath -o /out/server ./cmd/server

FROM alpine:3.20 AS base
RUN adduser -D -u 10001 hokm
USER hokm
COPY --from=build /out/server /server
EXPOSE 8080
ENTRYPOINT ["/server"]

FROM base AS prod
COPY --from=web /app/dist /web
ENV WEB_DIR=/web
