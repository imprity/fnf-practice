@echo off

set "to_search=D:\Apps\funkin_is_magic_ff2f7"

if "%~1"=="debug" (
	gdlv debug . -- "%to_search%"
)

if "%~1"=="" (
	go run . "%to_search%"
)
