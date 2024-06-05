#!/bin/bash

if [ "$1" == "" ]; then
    go build -tags=noaudio -gcflags="-e"
elif [ "$1" == "debug" ]; then
	go build -o "fnf-practice-debug" -tags=noaudio -gcflags="-e -l -N" .
else
    echo invalid arument "$1"
fi

