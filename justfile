set windows-shell := ["powershell.exe", "-NoLogo", "-Command"]

default: fmt vet test

fmt:
    go fmt ./...

vet:
    go vet ./...

test:
    go test ./...

build:
    go build ./...

install:
    go install .