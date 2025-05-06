# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOINSTALL=$(GOCMD) install
BINARY_NAME=go-novel-reader # 和 go.mod 中的模块名或者期望的二进制文件名一致
PKG_PATH=github.com/xqbumu/go-say # 确保这是你的模块路径

.PHONY: all build clean install run

all: build

# Builds the binary
build:
	@echo "Building $(BINARY_NAME)..."
	$(GOBUILD) -trimpath -ldflags "-s -w -extldflags '-static'" -o $(BINARY_NAME) main.go

# Installs the package
install:
	@echo "Installing $(PKG_PATH)..."
	$(GOINSTALL) $(PKG_PATH)@latest

# Cleans build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

# Runs the main program (useful for quick testing)
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BINARY_NAME) --help # 默认运行 help 命令
