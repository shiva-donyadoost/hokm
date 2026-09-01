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

function Test-HostPortBindable([int]$Port) {
    $listener = $null
    try {
        $listener = New-Object System.Net.Sockets.TcpListener ([System.Net.IPAddress]::Any, $Port)
        $listener.Start()
        return $true
    } catch {
        return $false
    } finally {
        if ($null -ne $listener) {
            try { $listener.Stop() } catch { }
        }
    }
}

function Select-BackendHostPort {
    $candidates = @(8080, 18080, 28080, 8000)
    foreach ($p in $candidates) {
        if (Test-HostPortBindable $p) { return $p }
    }
    Warn "no bindable backend host port among: $($candidates -join ', ')"
    Warn "check: netsh interface ipv4 show excludedportrange protocol=tcp"
    exit 1
}

$backendHostPort = Select-BackendHostPort
$env:BACKEND_HOST_PORT = "$backendHostPort"
if ($backendHostPort -ne 8080) {
    Warn "host port 8080 is reserved/in use; publishing backend on localhost:$backendHostPort"
}

Info "starting postgres + redis..."
Compose -ComposeArgs @("up", "-d", "postgres", "redis")

Info "rebuilding + starting backend (picks up Go code changes)..."
Compose -ComposeArgs @("up", "-d", "--build", "backend")

Info "starting frontend (vite hot reload watches ./frontend)..."
Compose -ComposeArgs @("up", "-d", "frontend")

Info "waiting for backend health..."
$healthy = $false
$healthUrl = "http://localhost:$backendHostPort/api/health"
for ($i = 0; $i -lt 60; $i++) {
    try {
        $resp = Invoke-RestMethod -Uri $healthUrl -TimeoutSec 2
        if ($resp.status -eq "ok") { $healthy = $true; break }
    } catch {
        Start-Sleep -Seconds 2
    }
}
if ($healthy) {
    Ok "backend healthy on http://localhost:$backendHostPort"
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
