param(
  [switch]$Rebuild,
  [switch]$NoBuild,
  [switch]$DryRun
)

$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
$desktopBinDir = Join-Path $repoRoot "cmd\anyclaw-desktop\build\bin"
$desktopExe = Join-Path $desktopBinDir "anyclaw-desktop.exe"
$buildScript = Join-Path $repoRoot "scripts\build-desktop.ps1"
$logDir = Join-Path $repoRoot ".anyclaw\logs"
$logFile = Join-Path $logDir "desktop-launch.log"

function Write-LauncherLog {
  param(
    [Parameter(Mandatory = $true)]
    [string]$Message
  )

  $line = "[{0}] {1}" -f (Get-Date -Format "yyyy-MM-dd HH:mm:ss"), $Message
  Write-Host $line
  Add-Content -LiteralPath $logFile -Value $line
}

function Invoke-LoggedScript {
  param(
    [Parameter(Mandatory = $true)]
    [string]$Path
  )

  & powershell.exe -NoLogo -NoProfile -ExecutionPolicy Bypass -File $Path *>&1 |
    Tee-Object -FilePath $logFile -Append

  if ($LASTEXITCODE -ne 0) {
    throw "Script failed with exit code $LASTEXITCODE`: $Path"
  }
}

New-Item -ItemType Directory -Force -Path $logDir | Out-Null
Set-Content -LiteralPath $logFile -Value ("AnyClaw Desktop launcher started at {0}" -f (Get-Date -Format "yyyy-MM-dd HH:mm:ss"))

try {
  Set-Location -LiteralPath $repoRoot

  $needsBuild = $Rebuild -or -not (Test-Path -LiteralPath $desktopExe -PathType Leaf)
  if ($needsBuild) {
    if ($NoBuild) {
      throw "Desktop executable not found: $desktopExe"
    }

    if (-not (Test-Path -LiteralPath $buildScript -PathType Leaf)) {
      throw "Desktop build script not found: $buildScript"
    }

    Write-LauncherLog "Desktop executable is missing or rebuild was requested. Building..."
    Invoke-LoggedScript -Path $buildScript
  }

  if (-not (Test-Path -LiteralPath $desktopExe -PathType Leaf)) {
    throw "Desktop build completed, but executable is missing: $desktopExe"
  }

  if ($DryRun) {
    Write-LauncherLog "Dry run completed. Desktop executable is ready: $desktopExe"
    return
  }

  Write-LauncherLog "Starting $desktopExe"
  Start-Process -FilePath $desktopExe -WorkingDirectory $desktopBinDir
  Write-LauncherLog "AnyClaw Desktop launch request was sent."
}
catch {
  Write-LauncherLog ("ERROR: {0}" -f $_.Exception.Message)
  Write-Error $_
  exit 1
}
