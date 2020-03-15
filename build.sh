#!/usr/bin/env sh
export GOOS=linux
export GOARCH=386
go build
zip go-fasthttp-sniproxy-chunks-linux-386.zip domains.txt domains-regex.txt go-fasthttp-sniproxy-chunks

export GOOS=linux
export GOARCH=amd64
go build
zip go-fasthttp-sniproxy-chunks-linux-amd64.zip domains.txt domains-regex.txt go-fasthttp-sniproxy-chunks

export GOOS=windows
export GOARCH=386
go build
zip go-fasthttp-sniproxy-chunks-win32.zip domains.txt domains-regex.txt go-fasthttp-sniproxy-chunks.exe

export GOOS=windows
export GOARCH=amd64
go build
zip go-fasthttp-sniproxy-chunks-win64.zip domains.txt domains-regex.txt go-fasthttp-sniproxy-chunks.exe
