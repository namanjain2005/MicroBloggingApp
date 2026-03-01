@echo off
echo ========================================
echo   Run Go tests (verbose) for repository
echo ========================================
@echo off
echo ========================================
echo   Run all Go tests (verbose)
echo ========================================

REM Ensure Go is available
go version >nul 2>&1
if %ERRORLEVEL% NEQ 0 (
	echo Go is not installed or not on PATH.
	exit /b 1
)

echo Running: go test -v ./...
go test -v ./...

echo.
echo ========================================
pause
