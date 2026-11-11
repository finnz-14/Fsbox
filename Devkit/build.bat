@echo off
setlocal
cd /d "%~dp0"

echo Building DevKit mod to main.wasm...
set GOOS=wasip1
set GOARCH=wasm
go build -buildmode=c-shared -o main.wasm .
if errorlevel 1 (
    echo Build failed.
    set GOOS=
    set GOARCH=
    exit /b 1
)
set GOOS=
set GOARCH=
echo Build complete: main.wasm
endlocal
