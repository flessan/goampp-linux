$ErrorActionPreference = 'Stop'
if (Get-Command gcc -ErrorAction SilentlyContinue) {
    Write-Host 'gcc is already available.'
    exit 0
}
if (-not (Get-Command choco -ErrorAction SilentlyContinue)) {
    throw 'Install MinGW-w64 gcc and add it to PATH (or install Chocolatey on a Windows runner).'
}
& choco install mingw -y --no-progress
if ($LASTEXITCODE -ne 0) { throw "MinGW installation failed: $LASTEXITCODE" }
$gccDir = 'C:\ProgramData\mingw64\mingw64\bin'
if (-not (Test-Path (Join-Path $gccDir 'gcc.exe'))) {
    throw "gcc.exe not found in $gccDir after installation; add your MinGW directory to PATH."
}
$env:PATH = "$gccDir;$env:PATH"
if ($env:GITHUB_PATH) { Add-Content -Path $env:GITHUB_PATH -Value $gccDir }
