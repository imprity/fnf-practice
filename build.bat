@echo off

if "%1" == "" (
	go build -o "fnf-practice.exe" -tags=noaudio -gcflags=all="-e" main.go
	goto :quit
)

if "%1"=="debug" (
	go build -o "fnf-practice-debug.exe" -tags=noaudio -gcflags=all="-e -l -N" main.go
	goto :quit
)

if "%1"=="font-gen" (
	go build -o "font-gen.exe" -tags=noaudio -gcflags=all="-e" font_gen.go
	goto :quit
)

if "%1"=="font-gen-debug" (
	go build -o "font-gen-debug.exe" -tags=noaudio -gcflags=all="-e -l -N" font_gen.go
	goto :quit
)

if "%1"=="all" (
	go build -o "fnf-practice.exe" -tags=noaudio -gcflags=all="-e" main.go || goto :quit
	go build -o "font-gen.exe" -tags=noaudio -gcflags=all="-e" font_gen.go
	goto :quit
)

echo invalid arument "%1"

:quit
