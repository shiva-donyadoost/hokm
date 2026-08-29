# Development-oriented multi-stage build (production hardening in Phase 18).
FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/server ./cmd/server

FROM alpine:3.20
RUN adduser -D -u 10001 hokm
USER hokm
COPY --from=build /out/server /server
EXPOSE 8080
ENTRYPOINT ["/server"]
