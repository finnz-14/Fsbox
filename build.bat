@echo off
setlocal

cd /d "%~dp0"

echo Building WASM mod...
pushd examplemod
set GOOS=wasip1
set GOARCH=wasm
go build -buildmode=c-shared -o ..\Mods\example.wasm .
if errorlevel 1 (
    echo Mod build failed
    popd
    exit /b 1
)
popd
set GOOS=
set GOARCH=

echo Building Go mod worker...
pushd ModsGo\example
go build -o bin\worker.exe .
if errorlevel 1 (
    echo Go mod worker build failed
    popd
    exit /b 1
)
popd

echo Building game...
go build -o bin\fsbox.exe .
if errorlevel 1 (
    echo Game build failed
    exit /b 1
)
copy /Y bin\fsbox.exe sandbox3d.exe >nul
copy /Y bin\fsbox.exe bin\sandbox3d.exe >nul

echo Build complete: bin\fsbox.exe, sandbox3d.exe
endlocal