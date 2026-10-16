@echo off
REM ============================================================
REM preflight.cmd — Windows wrapper for scripts/preflight.sh.
REM
REM Why this exists: on some machines `bash` on PATH resolves to
REM WSL bash (C:\Windows\System32\bash.exe), which cannot see Windows
REM processes or Docker Desktop and would report every port as free.
REM This wrapper always uses Git Bash.
REM
REM Usage:  scripts\preflight.cmd [--mode hybrid|docker|auto]
REM ============================================================
setlocal

set "GITBASH=%ProgramFiles%\Git\bin\bash.exe"
if not exist "%GITBASH%" set "GITBASH=%ProgramFiles(x86)%\Git\bin\bash.exe"
if not exist "%GITBASH%" set "GITBASH=%LOCALAPPDATA%\Programs\Git\bin\bash.exe"

if not exist "%GITBASH%" (
  echo [preflight] Git Bash not found. Install Git for Windows, or run:
  echo             bash scripts/preflight.sh
  exit /b 2
)

"%GITBASH%" "%~dp0preflight.sh" %*
exit /b %ERRORLEVEL%
