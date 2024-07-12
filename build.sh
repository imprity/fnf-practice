#!/bin/bash
set -e

if [ "$1" == "" ]; then

	go build -o "fnf-practice" -tags=noaudio -gcflags="-e" main.go

elif [ "$1" == "debug" ]; then

	go build -o "fnf-practice-debug" -tags=noaudio -gcflags="-e -l -N" main.go

elif [ "$1" == "font-gen" ]; then

	go build -o "font-gen" -tags=noaudio -gcflags="-e" font_gen.go

elif [ "$1" == "font-gen-debug" ]; then

	go build -o "font-gen-debug" -tags=noaudio -gcflags="-e -l -N" font_gen.go

elif [ "$1" == "all" ]; then

	go build -o "fnf-practice" -tags=noaudio -gcflags="-e" main.go
	go build -o "font-gen" -tags=noaudio -gcflags="-e" font_gen.go

else

    echo invalid arument "$1"

fi

