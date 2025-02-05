@echo off

set cmd=wikilite.exe --web --log

cd /d "%~dp0"
%cmd%
if %errorlevel% equ 0 (
    exit /b 0
)

cd ..
%cmd%
if %errorlevel% equ 0 (
    exit /b 0
)

exit /b 1
