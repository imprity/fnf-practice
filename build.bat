@echo off

SETLOCAL
SETLOCAL ENABLEDELAYEDEXPANSION

git describe --tags --always --abbrev=0 > nul

if !errorlevel! neq 0 (
	echo unknown > git_tag.txt
)

if !errorlevel! equ 0 (
	git describe --tags --always --abbrev=0 > git_tag.txt
)

if "%1" == "clean" (
	del fnf-practice.exe
	del fnf-practice-debug.exe
	del font-gen.exe
	del font-gen-debug.exe
	rmdir /S /Q release
	goto :quit
)

if "%1" == "" (
	set "to_build=fnf-practice.exe"
	set "build_source=main.go"
	call :build_command
	goto :quit
)

if "%1"=="font-gen" (
	set "to_build=font-gen.exe"
	set "build_source=font_gen.go"
	call :build_command
	goto :quit
)

if "%1"=="debug" (
	set "to_build=fnf-practice-debug.exe"
	set "build_source=main.go"
	call :build_debug_command
	goto :quit
)

if "%1"=="font-gen-debug" (
	set "to_build=font-gen-debug.exe"
	set "build_source=font_gen.go"
	call :build_debug_command
	goto :quit
)

if "%1"=="release" (
	set "to_build=fnf-practice.exe"
	set "build_source=main.go"
	call :build_command

	xcopy /Y .\fnf-practice.exe .\release\fnf-practice-win64\
	cd release && tar -cvzf fnf-practice-win64.zip .\fnf-practice-win64 & cd ..

	goto :quit
)

if "%1"=="all" (
	set "to_build=fnf-practice.exe"
	set "build_source=main.go"
	call :build_command

	if !errorlevel! neq 0 exit /b !errorlevel!

	set "to_build=font-gen.exe"
	set "build_source=font_gen.go"
	call :build_command

	if !errorlevel! neq 0 exit /b !errorlevel!

	xcopy /Y .\fnf-practice.exe .\release\fnf-practice-win64\
	cd release && tar -cvzf fnf-practice-win64.zip .\fnf-practice-win64 & cd ..

	goto :quit
)

echo invalid arument "%1"

goto :quit

:build_command
	echo building
	echo to_build : %to_build%
	echo build_source : %build_source%
	go build -o "%to_build%" -tags=noaudio -gcflags=all="-e" "%build_source%"
	exit /b !errorlevel!

:build_debug_command
	echo building debug
	echo to_build : %to_build%
	echo build_source : %build_source%
	go build -o "%to_build%" -tags=noaudio -gcflags=all="-e -l -N" "%build_source%"
	exit /b !errorlevel!

:quit

ENDLOCAL
