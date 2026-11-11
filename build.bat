@echo off
setlocal

cd /d "%~dp0"

echo Building game...
go build -o fsbox.exe .
if errorlevel 1 (
    echo Game build failed
    exit /b 1
)

echo Build complete: fsbox.exe
endlocal
