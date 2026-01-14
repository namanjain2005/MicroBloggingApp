@echo off
REM Quick Start Guide for Micro Blogging App (Windows)

echo ===== Micro Blogging App - Quick Start (Windows) =====
echo.

REM Check if Go is installed
go version >nul 2>&1
if %ERRORLEVEL% NEQ 0 (
    echo Go is not installed. Please install Go 1.25.2 or higher.
    exit /b 1
)

echo [OK] Go is installed: 
go version
echo.

REM Build server
echo Building server...
cd cmd\server
go build -o server.exe
if %ERRORLEVEL% NEQ 0 (
    echo [ERROR] Failed to build server
    exit /b 1
)
echo [OK] Server built successfully
cd ..\..
echo.

REM Build client
echo Building client...
cd cmd\client
go build -o client.exe
if %ERRORLEVEL% NEQ 0 (
    echo [ERROR] Failed to build client
    exit /b 1
)
echo [OK] Client built successfully
cd ..\..
echo.

echo ===== Setup Complete =====
echo.
echo Next steps:
echo.
echo 1. Start the server in one terminal:
echo    cd cmd\server
echo    .\server.exe
echo.
echo 2. Create a user in another terminal:
echo    cd cmd\client
echo    .\client.exe -cmd=create -name="Your Name" -password="YourPassword"
echo.
echo 3. Retrieve the user:
echo    cd cmd\client
echo    .\client.exe -cmd=get -id="^<user-id^>"
echo.
echo For more information, see README.md
echo.
pause
