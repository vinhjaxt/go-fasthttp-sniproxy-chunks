#!/usr/bin/env sh
# go get github.com/josephspurrier/goversioninfo/cmd/goversioninfo
echo 'Building linux i386..'
rm -f resource.syso
GOOS=linux GOARCH=386 go build
zip go-fasthttp-sniproxy-chunks-linux-386.zip domains.txt domains-regex.txt go-fasthttp-sniproxy-chunks

echo 'Building linux arm 32bit..'
rm -f resource.syso
GOOS=linux GOARCH=arm go build
zip go-fasthttp-sniproxy-chunks-linux-arm.zip domains.txt domains-regex.txt go-fasthttp-sniproxy-chunks

echo 'Building linux amd64..'
rm -f resource.syso
GOOS=linux GOARCH=amd64 go build
zip go-fasthttp-sniproxy-chunks-linux-amd64.zip domains.txt domains-regex.txt go-fasthttp-sniproxy-chunks

echo 'Building windows i386..'
rm -f resource.syso
goversioninfo -icon=icon.ico
GOOS=windows GOARCH=386 go build
zip go-fasthttp-sniproxy-chunks-win32.zip domains.txt domains-regex.txt go-fasthttp-sniproxy-chunks.exe

echo 'Building windows amd64..'
rm -f resource.syso
goversioninfo -icon=icon.ico
GOOS=windows GOARCH=amd64 go build
zip go-fasthttp-sniproxy-chunks-win64.zip domains.txt domains-regex.txt go-fasthttp-sniproxy-chunks.exe

echo 'Building macOS amd64..'
rm -f resource.syso
GOOS=darwin GOARCH=amd64 go build
zip go-fasthttp-sniproxy-chunks-macos.zip domains.txt domains-regex.txt go-fasthttp-sniproxy-chunks

rm -f resource.syso
