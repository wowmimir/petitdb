#!/usr/bin/env pwsh
# build.ps1 – PetitDB build helper for Windows / PowerShell

param(
    [string]$Task = "build",
    [string]$Port = "9379",
    [string]$Bind = "127.0.0.1",
    [string]$Dir = "./data",
    [string]$Version = $(git describe --tags --always --dirty 2>$null; if ($LASTEXITCODE -ne 0) { "dev" })
)

$BINARY = "petitdb.exe"
$MAIN_PKG = "./cmd/petitdb"

function Build {
    Write-Host "Building $BINARY (version $Version)..." -ForegroundColor Cyan
    go build -ldflags="-X main.Version=$Version" -o $BINARY $MAIN_PKG
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Build successful." -ForegroundColor Green
    } else {
        Write-Host "Build failed." -ForegroundColor Red
    }
}

function Test {
    Write-Host "Running tests with race detector..." -ForegroundColor Cyan
    go test -race -v ./...
    if ($LASTEXITCODE -eq 0) {
        Write-Host "All tests passed." -ForegroundColor Green
    } else {
        Write-Host "Tests failed." -ForegroundColor Red
    }
}

function Clean {
    Write-Host "Cleaning build artifacts and snapshots..." -ForegroundColor Cyan
    Remove-Item -Force -ErrorAction SilentlyContinue $BINARY
    Remove-Item -Force -ErrorAction SilentlyContinue snapshot, snapshot.tmp, snapshot.corrupt.*
    Remove-Item -Recurse -Force -ErrorAction SilentlyContinue data/
    Write-Host "Clean complete." -ForegroundColor Green
}

function Run {
    Write-Host "Starting server on $Bind`:$Port with data dir $Dir..." -ForegroundColor Cyan
    go run $MAIN_PKG --port $Port --bind $Bind --dir $Dir
}

function Fmt {
    Write-Host "Formatting code..." -ForegroundColor Cyan
    go fmt ./...
}

function Vet {
    Write-Host "Running go vet..." -ForegroundColor Cyan
    go vet ./...
}

function Help {
    @"
Available tasks:
  build   - Build the petitdb binary (default)
  test    - Run tests with race detector
  clean   - Remove binaries and temporary files
  run     - Run the server (customize with -Port, -Bind, -Dir)
  fmt     - Format source code
  vet     - Run go vet
  help    - Show this help

Examples:
  .\build.ps1 run -Port 9380 -Bind 0.0.0.0 -Dir C:\data
  .\build.ps1 test
  .\build.ps1 build -Version v1.2.3
"@
}

# Dispatch
switch ($Task) {
    "build" { Build }
    "test"  { Test }
    "clean" { Clean }
    "run"   { Run }
    "fmt"   { Fmt }
    "vet"   { Vet }
    "help"  { Help }
    default { Write-Host "Unknown task: $Task"; Help }
}