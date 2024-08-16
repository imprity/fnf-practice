#!/bin/bash

build_func () {
	echo "building"
	echo "to_build : $1"
	echo "build_source : $2"

	go build -o "$1" -tags=noaudio -gcflags=all="-e" "$2"
}

build_debug_func () {
	echo "building debug"
	echo "to_build : $1"
	echo "build_source : $2"

	go build -o "$1" -tags=noaudio -gcflags=all="-e -l -N" "$2"
}

build_demo_func () {
	echo "building demo"
	echo "to_build : fnf-practice-demo"
	echo "build_source : main.go"

    go build -o "fnf-practice-demo" \
        -tags=noaudio,demoreplay \
        -gcflags=all="-e" \
        "main.go"
}

git describe --tags --always --abbrev=0 > /dev/null
if [ $? -eq 0 ]; then
    git describe --tags --always --abbrev=0 > git_tag.txt
else
    echo "unknown" > git_tag.txt
fi

set -e

if [ "$1" == "" ]; then

    build_func fnf-practice main.go

elif [ "$1" == "clean" ]; then

    rm -f fnf-practice \
        fnf-practice-debug \
        font-gen \
        font-gen-debug \
        fnf-practice-demo

elif [ "$1" == "debug" ]; then

    build_debug_func fnf-practice-debug main.go

elif [ "$1" == "font-gen" ]; then

    build_func font-gen font_gen.go

elif [ "$1" == "font-gen-debug" ]; then

    build_debug_func font-gen-debug font_gen.go

elif [ "$1" == "demo" ]; then

    build_demo_func

elif [ "$1" == "all" ]; then

    build_func fnf-practice main.go
    build_func font-gen font_gen.go
    build_demo_func

else

    echo invalid arument "$1"

fi

