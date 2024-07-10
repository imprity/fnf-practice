@echo off

if "%1" == "" (
	go build -o "fnf-practice.exe" -tags=noaudio -gcflags="-e" main.go
	goto :quit
)

if "%1"=="debug" (
	go build -o "fnf-practice-debug.exe" -tags=noaudio -gcflags="-e -l -N" main.go
	goto :quit
)

echo invalid arument "%1"

:quit
