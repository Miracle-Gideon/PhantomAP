@echo off
echo.
echo   PhantomAP :: Build Receiver
echo   Built Miracle Gideon
echo.

where go >nul 2>&1
if %errorlevel% neq 0 (
    echo   ERR :: Go not found. Download from https://golang.org/dl/
    pause
    exit /b 1
)

cd /d "%~dp0receiver"
go build -ldflags="-s -w" -o ..\phantomAP_receiver.exe .
if %errorlevel% neq 0 (
    echo   ERR :: Build failed.
    pause
    exit /b 1
)

cd ..
echo   BUILD OK :: phantomAP_receiver.exe
echo.
pause
