@echo off
setlocal

cd /d "%~dp0"

set TAGS=
if /I "%~1"=="release" set TAGS=-tags release

echo Building dedicated server...
if defined TAGS (
    echo Release build: smoke test code excluded
)
go build %TAGS% -o fsbox-server.exe ..
if errorlevel 1 (
    echo Dedicated server build failed
    exit /b 1
)

echo Build complete: server\fsbox-server.exe
endlocal
