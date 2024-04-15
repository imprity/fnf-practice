@echo off

go build -tags noaudio . && gofmt -w -s .
