@echo off

SETLOCAL
SETLOCAL ENABLEDELAYEDEXPANSION

git describe --tags --always --abbrev=0 > nul

if !errorlevel! neq 0 (
	echo could not get git tag!
	echo unknown > git_tag.txt
)

if !errorlevel! equ 0 (
	git describe --tags --always --abbrev=0 > git_tag.txt
)

set /P git_ver_str=<git_tag.txt
echo git_ver_str %git_ver_str%

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
	if !errorlevel! neq 0 exit /b !errorlevel!

	call :create_release
	if !errorlevel! neq 0 exit /b !errorlevel!

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

	call :create_release
	if !errorlevel! neq 0 exit /b !errorlevel!

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

:create_release
	rmdir /S /Q release
	if exist release (
		echo Failed to delete release folder!
		exit /b 1
	)	

	set "release_folder=.\release\fnf-practice-win64-v%git_ver_str%"

	xcopy /Y .\fnf-practice.exe "%release_folder%\"
	powershell Compress-Archive -Force^
		"%release_folder%"^
		"%release_folder%.zip"
		
	exit /b !errorlevel!

:quit

ENDLOCAL
