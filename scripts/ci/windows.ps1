param(
    [ValidateSet('format', 'test', 'build', 'all')]
    [string]$Mode = 'all'
)
$ErrorActionPreference = 'Stop'
Set-Location (Join-Path $PSScriptRoot '../..')
foreach ($cmd in @('go', 'gcc')) {
    if (-not (Get-Command $cmd -ErrorAction SilentlyContinue)) { throw "Required command not found: $cmd" }
}
$env:CGO_ENABLED = '1'
function Invoke-Go([string[]]$Arguments) {
    & go @Arguments
    if ($LASTEXITCODE -ne 0) { throw "go $Arguments failed with exit code $LASTEXITCODE" }
}
function Test-Format {
    if (-not (Get-Command git -ErrorAction SilentlyContinue)) { throw 'git is required' }
    if ($env:GOAMPP_FORMAT_BASE) {
        & git rev-parse --verify "$($env:GOAMPP_FORMAT_BASE)^{commit}" | Out-Null
        if ($LASTEXITCODE -ne 0) { throw 'Invalid GOAMPP_FORMAT_BASE' }
        $files = @(git diff --name-only --diff-filter=ACMRT $env:GOAMPP_FORMAT_BASE -- '*.go')
        if ($LASTEXITCODE -ne 0) { throw 'Could not list changed Go files' }
        $files += @(git ls-files --others --exclude-standard -- '*.go')
    } else {
        $files = @(git ls-files -- '*.go')
    }
    if ($LASTEXITCODE -ne 0) { throw 'Could not list Go files' }
    if ($files.Count -eq 0) { Write-Host 'No Go files to check'; return }
    $unformatted = @(gofmt -l $files)
    if ($LASTEXITCODE -ne 0) { throw 'gofmt failed' }
    if ($unformatted.Count -gt 0) { throw "Go files needing formatting:`n$($unformatted -join "`n")" }
}
function Test-Go { Invoke-Go -Arguments @('vet', './...'); Invoke-Go -Arguments @('test', './...', '-count=1') }
function Build-Go {
    New-Item -ItemType Directory -Force dist/windows | Out-Null
    Invoke-Go -Arguments @('build', '-trimpath', '-o', 'dist/windows/goampp.exe', '.')
}
switch ($Mode) {
    'format' { Test-Format }
    'test' { Test-Go }
    'build' { Build-Go }
    'all' { Test-Format; Test-Go; Build-Go }
}
