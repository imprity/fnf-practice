@echo off

if "%1" == "" (
	go build -tags=noaudio -gcflags="-e" .
	goto :quit
)

if "%1"=="debug" (
	go build -o "fnf-practice-debug.exe" -tags=noaudio -gcflags="-e -l -N" .
	goto :quit
)

echo invalid arument "%1"

:quit
