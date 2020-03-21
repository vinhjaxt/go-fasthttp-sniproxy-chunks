#!/usr/bin/env sh
# go get github.com/josephspurrier/goversioninfo/cmd/goversioninfo
rm -f resource.syso
GOOS=linux GOARCH=386 go build
zip go-fasthttp-sniproxy-chunks-linux-386.zip domains.txt domains-regex.txt go-fasthttp-sniproxy-chunks

rm -f resource.syso
GOOS=linux GOARCH=amd64 go build
zip go-fasthttp-sniproxy-chunks-linux-amd64.zip domains.txt domains-regex.txt go-fasthttp-sniproxy-chunks

rm -f resource.syso
goversioninfo -icon=icon.ico
GOOS=windows GOARCH=386 go build
zip go-fasthttp-sniproxy-chunks-win32.zip domains.txt domains-regex.txt go-fasthttp-sniproxy-chunks.exe

rm -f resource.syso
goversioninfo -icon=icon.ico
GOOS=windows GOARCH=amd64 go build
zip go-fasthttp-sniproxy-chunks-win64.zip domains.txt domains-regex.txt go-fasthttp-sniproxy-chunks.exe

rm -f resource.syso
