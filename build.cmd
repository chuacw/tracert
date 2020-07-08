@@echo off
setlocal
set GOPATH=%CD%
set GOOS=linux
set GOARCH=arm
go get "golang.org/x/net/icmp"
go get "golang.org/x/net/ipv4"
go get "golang.org/x/net/ipv6"
go build
