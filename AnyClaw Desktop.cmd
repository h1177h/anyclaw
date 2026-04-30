@echo off
setlocal

set "ROOT=%~dp0"
powershell.exe -NoLogo -NoProfile -ExecutionPolicy Bypass -File "%ROOT%scripts\start-desktop.ps1" %*
set "STATUS=%ERRORLEVEL%"

if not "%STATUS%"=="0" (
  echo.
  echo AnyClaw Desktop did not start. Exit code %STATUS%.
  echo See .anyclaw\logs\desktop-launch.log for details when available.
  echo.
  pause
)

exit /b %STATUS%
