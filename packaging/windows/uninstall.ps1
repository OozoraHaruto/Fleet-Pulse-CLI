param(
  [string]$InstallDir = "$env:ProgramFiles\FleetPulse",
  [string]$StateDir = "$env:ProgramData\FleetPulse",
  [string]$ServiceName = "FleetPulse",
  [switch]$Purge
)

$ErrorActionPreference = "Stop"

sc.exe stop $ServiceName | Out-Null 2>$null
sc.exe delete $ServiceName | Out-Null 2>$null
Remove-Item -Force -ErrorAction SilentlyContinue (Join-Path $InstallDir "fleetpulse.exe")

if ($Purge) {
  Remove-Item -Recurse -Force -ErrorAction SilentlyContinue $InstallDir
  Remove-Item -Recurse -Force -ErrorAction SilentlyContinue $StateDir
  Write-Host "FleetPulse uninstalled and state purged."
} else {
  Write-Host "FleetPulse uninstalled. Preserved $StateDir."
}
