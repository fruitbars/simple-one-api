.PHONY: release build

release: build upx compress

dev: build

build:
	@$(MAKE) --no-print-directory \
    build-darwin-amd64 build-darwin-arm64 \
    build-windows-amd64 build-windows-arm64 \
    build-linux-amd64 build-linux-arm64

build-darwin-amd64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o build/darwin-amd64/simpleoneapi

build-darwin-arm64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o build/darwin-arm64/simpleoneapi

build-windows-amd64:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o build/windows-amd64/simpleoneapi.exe

build-windows-arm64:
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -o build/windows-arm64/simpleoneapi.exe

build-linux-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/linux-amd64/simpleoneapi

build-linux-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o build/linux-arm64/simpleoneapi

upx:
	upx --best --lzma build/darwin-amd64/simpleoneapi; \
    upx --best --lzma build/windows-amd64/simpleoneapi.exe; \
    upx --best --lzma build/linux-amd64/simpleoneapi;

compress:
	@$(MAKE) --no-print-directory \
    compress-darwin-amd64 compress-darwin-arm64 \
    compress-windows-amd64 compress-windows-arm64 \
    compress-linux-amd64 compress-linux-arm64

compress-darwin-amd64:
	tar -czvf build/wxbot-darwin-amd64.tar.gz -C build/darwin-amd64/ simpleoneapi; \
    rm -rf build/darwin-amd64

compress-darwin-arm64:
	tar -czvf build/wxbot-darwin-arm64.tar.gz -C build/darwin-arm64/ simpleoneapi; \
    rm -rf build/darwin-arm64

compress-windows-amd64:
	tar -czvf build/wxbot-windows-amd64.tar.gz -C build/windows-amd64/ simpleoneapi.exe; \
    rm -rf build/windows-amd64

compress-windows-arm64:
	tar -czvf build/wxbot-windows-arm64.tar.gz -C build/windows-arm64/ simpleoneapi.exe; \
    rm -rf build/windows-arm64

compress-linux-amd64:
	tar -czvf build/wxbot-linux-amd64.tar.gz -C build/linux-amd64/ simpleoneapi; \
    rm -rf build/linux-amd64

compress-linux-arm64:
	tar -czvf build/wxbot-linux-arm64.tar.gz -C build/linux-arm64/ simpleoneapi; \
    rm -rf build/linux-arm64
