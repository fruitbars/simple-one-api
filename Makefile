.PHONY: release build compress clean

BINARY_NAME=simple-one-api
BUILD_DIR=build
use_upx?=0  # 设置默认值为0，表示默认禁用 UPX 压缩
clean_up?=0  # 默认不删除构建目录

PLATFORMS = darwin-amd64 darwin-arm64 windows-amd64 windows-arm64 linux-amd64 linux-arm64 freebsd-amd64 freebsd-arm64

release: build upx compress

dev: build

build: $(PLATFORMS)

$(PLATFORMS):
	@$(MAKE) --no-print-directory build-$@

build-darwin-amd64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/darwin-amd64/$(BINARY_NAME)

build-darwin-arm64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/darwin-arm64/$(BINARY_NAME)

build-windows-amd64:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/windows-amd64/$(BINARY_NAME).exe

build-windows-arm64:
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -o $(BUILD_DIR)/windows-arm64/$(BINARY_NAME).exe

build-linux-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/linux-amd64/$(BINARY_NAME)

build-linux-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o $(BUILD_DIR)/linux-arm64/$(BINARY_NAME)

build-freebsd-amd64:
	CGO_ENABLED=0 GOOS=freebsd GOARCH=amd64 go build -o $(BUILD_DIR)/freebsd-amd64/$(BINARY_NAME)

build-freebsd-arm64:
	CGO_ENABLED=0 GOOS=freebsd GOARCH=arm64 go build -o $(BUILD_DIR)/freebsd-arm64/$(BINARY_NAME)

upx:
ifeq ($(use_upx),1)
	upx --best --lzma $(BUILD_DIR)/darwin-amd64/$(BINARY_NAME); \
    upx --best --lzma $(BUILD_DIR)/windows-amd64/$(BINARY_NAME).exe; \
    upx --best --lzma $(BUILD_DIR)/linux-amd64/$(BINARY_NAME);\
    upx --best --lzma $(BUILD_DIR)/freebsd-amd64/$(BINARY_NAME);\
    upx --best --lzma $(BUILD_DIR)/darwin-arm64/$(BINARY_NAME); \
    upx --best --lzma $(BUILD_DIR)/windows-arm64/$(BINARY_NAME).exe; \
    upx --best --lzma $(BUILD_DIR)/linux-arm64/$(BINARY_NAME);	\
    upx --best --lzma $(BUILD_DIR)/freebsd-arm64/$(BINARY_NAME)
endif

compress:
	@$(MAKE) --no-print-directory $(addprefix compress-,$(PLATFORMS))

compress-darwin-amd64:
	cp -r static $(BUILD_DIR)/darwin-amd64/
	tar -czvf $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64.tar.gz -C $(BUILD_DIR)/darwin-amd64/ $(BINARY_NAME) static
ifeq ($(clean_up),1)
	rm -rf $(BUILD_DIR)/darwin-amd64
	@echo "Removed build directory for darwin-amd64"
endif

compress-darwin-arm64:
	cp -r static $(BUILD_DIR)/darwin-arm64/
	tar -czvf $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64.tar.gz -C $(BUILD_DIR)/darwin-arm64/ $(BINARY_NAME) static
ifeq ($(clean_up),1)
	rm -rf $(BUILD_DIR)/darwin-arm64
	@echo "Removed build directory for darwin-arm64"
endif

compress-windows-amd64:
	cp -r static $(BUILD_DIR)/windows-amd64/
	zip -j $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.zip $(BUILD_DIR)/windows-amd64/$(BINARY_NAME).exe
	zip -r $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.zip $(BUILD_DIR)/windows-amd64/static
ifeq ($(clean_up),1)
	rm -rf $(BUILD_DIR)/windows-amd64
	@echo "Removed build directory for windows-amd64"
endif

compress-windows-arm64:
	cp -r static $(BUILD_DIR)/windows-arm64/
	zip -j $(BUILD_DIR)/$(BINARY_NAME)-windows-arm64.zip $(BUILD_DIR)/windows-arm64/$(BINARY_NAME).exe
	zip -r $(BUILD_DIR)/$(BINARY_NAME)-windows-arm64.zip $(BUILD_DIR)/windows-arm64/static
ifeq ($(clean_up),1)
	rm -rf $(BUILD_DIR)/windows-arm64
	@echo "Removed build directory for windows-arm64"
endif

compress-linux-amd64:
	cp -r static $(BUILD_DIR)/linux-amd64/
	tar -czvf $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64.tar.gz -C $(BUILD_DIR)/linux-amd64/ $(BINARY_NAME) static
ifeq ($(clean_up),1)
	rm -rf $(BUILD_DIR)/linux-amd64
	@echo "Removed build directory for linux-amd64"
endif

compress-linux-arm64:
	cp -r static $(BUILD_DIR)/linux-arm64/
	tar -czvf $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64.tar.gz -C $(BUILD_DIR)/linux-arm64/ $(BINARY_NAME) static
ifeq ($(clean_up),1)
	rm -rf $(BUILD_DIR)/linux-arm64
	@echo "Removed build directory for linux-arm64"
endif

compress-freebsd-amd64:
	cp -r static $(BUILD_DIR)/freebsd-amd64/
	tar -czvf $(BUILD_DIR)/$(BINARY_NAME)-freebsd-amd64.tar.gz -C $(BUILD_DIR)/freebsd-amd64/ $(BINARY_NAME) static
ifeq ($(clean_up),1)
	rm -rf $(BUILD_DIR)/freebsd-amd64
	@echo "Removed build directory for freebsd-amd64"
endif

compress-freebsd-arm64:
	cp -r static $(BUILD_DIR)/freebsd-arm64/
	tar -czvf $(BUILD_DIR)/$(BINARY_NAME)-freebsd-arm64.tar.gz -C $(BUILD_DIR)/freebsd-arm64/ $(BINARY_NAME) static
ifeq ($(clean_up),1)
	rm -rf $(BUILD_DIR)/freebsd-arm64
	@echo "Removed build directory for freebsd-arm64"
endif

clean:
	rm -rf $(BUILD_DIR)/*