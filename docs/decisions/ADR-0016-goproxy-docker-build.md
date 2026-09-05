# ADR-0016: Go module proxy for Docker builds

- Status: Accepted
- Date: 2026-09-05

## Context
proxy.golang.org TLS handshake times out in Docker builds on this network.

## Decision
Set GOPROXY=https://goproxy.cn,direct in the Go build stage Dockerfile.

## Consequences
Builds use a mirror first; direct fallback remains.
