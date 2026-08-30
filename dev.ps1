# dev.ps1 - development launcher for the HOKM platform.
#
# Usage:
#   .\dev.ps1              start postgres/redis, rebuild + start backend,
#                          start frontend with hot reload, open the browser
#   .\dev.ps1 -NoBrowser   same, but do not open the browser
#   .\dev.ps1 -Down        stop the whole development stack
#
# Development model:
#   - frontend: vite dev server in Docker bind-mounting ./frontend ->
#     code changes hot-reload automatically (chokidar polling enabled).
#   - backend: Go image rebuilt from ./backend -> re-run this script (or
#     "docker compose up -d --build backend") after changing Go code.

param(
    [switch]$Down,
    [switch]$NoBrowser
)

$ErrorActionPreference = "Stop"
Set-Location -LiteralPath $PSScriptRoot

function Info($msg) { Write-Host "[dev] $msg" -ForegroundColor Cyan }
function Ok($msg)   { Write-Host "[ok ] $msg" -ForegroundColor Green }
function Warn($msg) { Write-Host "[!]  $msg" -ForegroundColor Yellow }

# Run docker compose and abort the script on failure - never keep serving
# a stale container after a failed build.
function Compose {
    param([string[]]$ComposeArgs)
    docker compose @ComposeArgs
    if ($LASTEXITCODE -ne 0) {
        Warn "docker compose $($ComposeArgs -join ' ') FAILED - see output above."
        exit 1
    }
}

if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
    Warn "docker not found - start Docker Desktop and retry."
    exit 1
}

if ($Down) {
    Info "stopping the development stack..."
    docker compose down
    exit 0
}

if (-not (Test-Path ".env")) {
    Copy-Item ".env.example" ".env"
    Info "created .env from .env.example (dev defaults)"
}

Info "starting postgres + redis..."
Compose -ComposeArgs @("up", "-d", "postgres", "redis")

Info "rebuilding + starting backend (picks up Go code changes)..."
Compose -ComposeArgs @("up", "-d", "--build", "backend")

Info "starting frontend (vite hot reload watches ./frontend)..."
Compose -ComposeArgs @("up", "-d", "frontend")

Info "waiting for backend health..."
$healthy = $false
for ($i = 0; $i -lt 60; $i++) {
    try {
        $resp = Invoke-RestMethod -Uri "http://localhost:8080/api/health" -TimeoutSec 2
        if ($resp.status -eq "ok") { $healthy = $true; break }
    } catch {
        Start-Sleep -Seconds 2
    }
}
if ($healthy) {
    Ok "backend healthy on http://localhost:8080"
} else {
    Warn "backend not healthy yet - check: docker compose logs backend"
}

Write-Host ""
docker compose ps
Write-Host ""

$fe = "http://localhost:5173"
Ok "HOKM dev running: $fe"
Write-Host "      frontend changes  -> hot reload (no action needed)"
Write-Host "      backend changes   -> re-run this script to rebuild"
if (-not $NoBrowser) { Start-Process $fe }
