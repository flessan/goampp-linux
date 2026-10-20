@echo off
REM Build goampp.exe: a 64-bit Windows GUI app with the icon, version info and
REM manifest compiled in.
REM Usage: build.bat [version]   (default 0.0.0)
REM
REM The Fyne GUI needs cgo, so a MinGW-w64 gcc must be on PATH
REM (e.g. winget install BrechtSanders.WinLibs.POSIX.UCRT, or MSYS2 mingw64).
setlocal
set VERSION=%~1
if "%VERSION%"=="" set VERSION=0.0.0

set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=1

where gcc >nul 2>nul || (
    echo gcc not found on PATH - install MinGW-w64 first.
    exit /b 1
)

echo Embedding icon and version info...
go run github.com/tc-hib/go-winres@v0.3.3 simply --arch amd64 --manifest gui ^
    --icon assets\logo.png --product-name GoAMPP --file-description "GoAMPP local web stack" ^
    --original-filename goampp.exe --product-version %VERSION% --file-version %VERSION%
if errorlevel 1 exit /b 1

echo Building goampp.exe v%VERSION%...
go build -ldflags="-H windowsgui -s -w -X goampp/internal/core.Version=v%VERSION%" -trimpath -o goampp.exe .
if errorlevel 1 (
    echo Build FAILED.
    exit /b 1
)

echo.
echo Build OK:
dir goampp.exe | findstr goampp.exe
endlocal
