param([string]$OutputDirectory = 'snapshots/windows')
$ErrorActionPreference = 'Stop'
Set-Location (Join-Path $PSScriptRoot '../..')
if (-not (Get-Command go -ErrorAction SilentlyContinue)) { throw 'Go is required for Windows captures' }
New-Item -ItemType Directory -Force $OutputDirectory | Out-Null
$env:GOAMPP_SNAPSHOT = (Resolve-Path $OutputDirectory).Path
foreach ($variant in @('Light', 'Dark', 'System')) {
    foreach ($page in @('services', 'projects', 'vhosts', 'editor', 'startup', 'settings')) {
        $file = Join-Path $env:GOAMPP_SNAPSHOT "${variant}_${page}.png"
        if (Test-Path $file) { Remove-Item $file -Force }
    }
}
& go test ./internal/gui -run '^TestGUISnapshots$' -count=1
if ($LASTEXITCODE -ne 0) { throw "Windows snapshot test failed with exit code $LASTEXITCODE" }
foreach ($variant in @('Light', 'Dark')) {
    foreach ($page in @('services', 'projects', 'vhosts', 'editor', 'startup', 'settings')) {
        $file = Join-Path $env:GOAMPP_SNAPSHOT "${variant}_${page}.png"
        if (-not (Test-Path $file) -or (Get-Item $file).Length -eq 0) { throw "Missing screenshot: $file" }
    }
}
Write-Host "Windows screenshots written to $env:GOAMPP_SNAPSHOT"
