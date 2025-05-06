@echo off
title OMSAY Launcher

echo Starting OMSAY Server...
start cmd /k "go run server\main.go"

timeout /t 2 /nobreak >nul

echo Launching OMSAY Client...
start client\omsay.exe

exit