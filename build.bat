@echo off
setlocal

cd /d "%~dp0"

echo Building WASM mods...
for /d %%d in (Mods\*) do (
    if not exist "%%d\go.mod" (
        if exist "%%d\main.go" (
            echo Building WASM mod %%d...
            pushd %%d
            set GOOS=wasip1
            set GOARCH=wasm
            go build -buildmode=c-shared -o main.wasm .
            if errorlevel 1 (
                echo Mod build failed for %%d
                popd
                exit /b 1
            )
            popd
            set GOOS=
            set GOARCH=
        )
    )
)

echo Building Go mod workers...
for /d %%d in (Mods\*) do (
    if exist "%%d\go.mod" (
        echo Building Go mod worker %%d...
        if not exist "%%d\bin" mkdir "%%d\bin"
        pushd %%d
        go build -o bin\worker.exe .
        if errorlevel 1 (
            echo Go mod worker build failed for %%d
            popd
            exit /b 1
        )
        popd
    )
)

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
