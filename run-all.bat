@echo off
setlocal
cd /d %~dp0

if not exist cmd\server\server.exe (
    echo [ERROR] cmd\server\server.exe not found. Run quickstart.bat first.
    exit /b 1
)
if not exist cmd\timeline-consumer\timeline-consumer.exe (
    echo [ERROR] cmd\timeline-consumer\timeline-consumer.exe not found. Run quickstart.bat first.
    exit /b 1
)

start "" cmd\server\server.exe
start "" cmd\timeline-consumer\timeline-consumer.exe

echo [OK] Server and timeline-consumer started.
echo Close the opened windows to stop them.
