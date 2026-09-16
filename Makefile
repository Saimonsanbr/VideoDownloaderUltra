BIN=videodownloaderultra
PKG=github.com/Saimonsanbr/VideoDownloaderUltra

.PHONY: deps run gui build build-all clean tidy

deps:
	go mod tidy

run:
	go run ./cmd/videodownloaderultra --help

gui:
	go run ./cmd/videodownloaderultra --gui

build:
	go build -ldflags="-s -w" -o $(BIN) ./cmd/videodownloaderultra

build-all:
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o dist/$(BIN)-darwin-arm64 ./cmd/videodownloaderultra
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o dist/$(BIN)-darwin-amd64 ./cmd/videodownloaderultra
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o dist/$(BIN)-linux-amd64 ./cmd/videodownloaderultra
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o dist/$(BIN)-linux-arm64 ./cmd/videodownloaderultra
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o dist/$(BIN)-windows-amd64.exe ./cmd/videodownloaderultra
	ls -lh dist/

tidy:
	go mod tidy

clean:
	rm -rf dist $(BIN) *.mp4 *.webm
