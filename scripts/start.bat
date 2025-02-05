@echo off

set cmd=wikilite.exe --log --setup --web

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
