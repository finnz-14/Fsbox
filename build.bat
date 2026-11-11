@echo off
setlocal

cd /d "%~dp0"

set TAGS=
if /I "%~1"=="release" set TAGS=-tags release

echo Building game...
if defined TAGS (
    echo Release build: smoke test code excluded
)
go build %TAGS% -o fsbox.exe .
if errorlevel 1 (
    echo Game build failed
    exit /b 1
)

echo Build complete: fsbox.exe
endlocal
