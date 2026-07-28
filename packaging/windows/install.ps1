param(
  [string]$BinarySource = ".\fleetpulse.exe",
  [string]$InstallDir = "$env:ProgramFiles\FleetPulse",
  [string]$StateDir = "$env:ProgramData\FleetPulse",
  [string]$ServiceName = "FleetPulse",
  [switch]$StartService
)

$ErrorActionPreference = "Stop"

if (!(Test-Path $BinarySource)) {
  throw "fleetpulse binary not found at $BinarySource"
}

New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
New-Item -ItemType Directory -Force -Path $StateDir | Out-Null
Copy-Item $BinarySource -Destination (Join-Path $InstallDir "fleetpulse.exe") -Force

$ConfigPath = Join-Path $StateDir "fleetpulse.json"
$TokenPath = Join-Path $StateDir "token"
if (!(Test-Path $ConfigPath)) {
  @{
    addr = "0.0.0.0:35338"
    auth_enabled = $true
    token_file = $TokenPath
    cache_ttl = "10s"
    collector_timeout = "2s"
    log_level = "info"
    service_name = "fleetpulse"
    deployment_target = "windows"
  } | ConvertTo-Json | Set-Content -Path $ConfigPath -Encoding UTF8
}

& (Join-Path $InstallDir "fleetpulse.exe") token show -token-file $TokenPath *> $null
if ($LASTEXITCODE -ne 0) {
  & (Join-Path $InstallDir "fleetpulse.exe") token rotate -token-file $TokenPath | Out-Null
}

$BinaryPath = "`"$InstallDir\fleetpulse.exe`" serve -config `"$ConfigPath`""
sc.exe stop $ServiceName | Out-Null 2>$null
sc.exe delete $ServiceName | Out-Null 2>$null
sc.exe create $ServiceName binPath= $BinaryPath start= auto DisplayName= "FleetPulse telemetry agent" | Out-Null

if ($StartService) {
  sc.exe start $ServiceName | Out-Null
}

Write-Host "FleetPulse installed. Token is stored at $TokenPath."
