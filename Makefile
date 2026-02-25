build-windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o build/aradel-pi.exe cmd/main.go