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

# Rebuild the embedded browse UI from frontend/ into internal/browse/web/.
# Run this whenever the frontend source changes; it is never part of `go build`.
build-web:
    cd frontend && npm ci && npm run build