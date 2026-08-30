@echo off
rem start.bat — double-click launcher for the HOKM platform (dev stack).
cd /d "%~dp0"

where docker >nul 2>nul
if errorlevel 1 (
  echo ERROR: docker not found - start Docker Desktop first.
  pause
  exit /b 1
)

if not exist .env copy .env.example .env >nul

echo building and starting containers...
docker compose up --build -d

echo waiting for the backend to become healthy...
set /a tries=0
:waitloop
curl -sf http://localhost:8080/api/health >nul 2>nul
if %errorlevel%==0 goto ready
set /a tries+=1
if %tries% geq 60 goto timeout
timeout /t 2 /nobreak >nul
goto waitloop

:timeout
echo WARNING: backend not healthy yet - check: docker compose logs backend

:ready
echo.
echo   HOKM is running:  http://localhost:5173
echo   stop:             docker compose down
echo.
start "" http://localhost:5173
docker compose ps
pause
