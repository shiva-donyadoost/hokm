# ADR-0007: Frontend Framework

Date: 2026-08-29
Status: Accepted

## Context

Mobile-first real-time game UI, 2–4 authenticated views, heavy custom
rendering of a card table, animations, and touch interactions.

## Decision

**React 18 + TypeScript + Vite**, no UI framework beyond Tailwind CSS for
layout utilities. Game table is custom DOM/CSS (cards are absolutely
positioned elements animated with CSS transitions); no canvas needed at
this complexity.

- State: server is authority; client keeps a view-model reducer fed by WS
  events (mirrors ADR-0004 views). Zustand for global stores (small,
  ergonomic); React Query only if REST caching grows beyond auth/profile.
- Build: Vite (fast HMR, ESM output). ESLint + Prettier enforced.
- i18n-ready: strings centralized from day one (fa/en), RTL layout support
  via logical CSS properties.

## Alternatives considered

- **Vue/Svelte**: viable, but React has the deepest ecosystem for realtime
  UI patterns and the team's existing knowledge; decision cost > benefit.
- **Canvas/WebGL (Phaser)**: overkill for a trick-taking card game; DOM
  cards are accessible (a11y) and easier to make responsive.
- **Next.js**: SSR/SEO adds complexity irrelevant to an authenticated game
  client; static SPA + API is simpler to deploy behind the Go server.

## Consequences

- `frontend/src/protocol/` mirrors the Go WS envelope (types + helpers).
- Static build is served by the Go backend in production (single deployable
  artifact); dev uses Vite proxy to `localhost:8080`.
