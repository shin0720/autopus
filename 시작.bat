@echo off
cd /d "%~dp0"

echo 포트 8080 정리 중...
for /f "tokens=5" %%a in ('netstat -ano ^| findstr ":8080 " ^| findstr "LISTENING"') do (
    taskkill /PID %%a /F >nul 2>&1
)

echo Autopus Studio 시작 중...
start /min "" "%~dp0AutopusStudio.exe" ui
exit
