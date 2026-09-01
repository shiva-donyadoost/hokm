# ADR-0011: Windows Host Port Remap for Docker Publish

Date: 2026-08-31
Status: Accepted

## Context

`.\dev.ps1` failed while starting the backend container:

```
ports are not available: exposing port TCP 0.0.0.0:8080
listen tcp 0.0.0.0:8080: bind: An attempt was made to access a socket
in a way forbidden by its access permissions.
```

Nothing was listening on 8080. `netsh interface ipv4 show excludedportrange
protocol=tcp` showed Hyper-V/WinNAT had reserved 8061-8160, which includes
8080. That exclusion is dynamic across reboots; "address already in use"
is the wrong diagnosis.

The container still listens on `APP_ADDR=:8080`. Only the **host publish**
port is blocked. The Vite proxy talks to `backend:8080` on the compose
network, so the browser path through `:5173` does not depend on the host
mapping. The launcher health check and direct API curls do.

There is no `dev.bat`; Windows users who double-click or type `dev.bat`
get a missing-file error.

## Decision

- Keep the process listen address at `:8080` inside the container.
- Publish the host mapping as `${BACKEND_HOST_PORT:-8080}:8080`.
- `dev.ps1` probes bindability on 0.0.0.0 (TcpListener) and selects the
  first free candidate: 8080, 18080, 28080, 8000. It exports
  `BACKEND_HOST_PORT` for compose interpolation and uses that port for
  the host health check.
- Add `dev.bat` as a thin wrapper that invokes `dev.ps1` with the same
  arguments (`-Down`, `-NoBrowser`).

## Alternatives considered

- **Restart WinNAT / reserve 8080 via netsh**: rejected — needs
  Administrator, and Hyper-V re-reserves ranges on the next boot.
- **Hard-code host 18080 forever**: rejected — machines that can bind
  8080 should keep the documented default.
- **Change `APP_ADDR` / Vite proxy to the host port**: rejected — those
  are container-network addresses; remapping the host publish is enough.

## Consequences

- Direct host access to the API may be `localhost:18080` (or another
  candidate) on Windows hosts with 8080 excluded. The UI on `:5173` is
  unchanged.
- Raw `docker compose up` without the launcher still uses 8080 unless
  `BACKEND_HOST_PORT` is set in the environment or `.env`.
