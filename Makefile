.PHONY: help build build-all install test lint fmt clean run
.PHONY: build-linux build-linux-loong64 build-linux-musl build-darwin build-windows build-freebsd
.PHONY: dist dist-linux dist-darwin dist-windows dist-tarball dist-zip
.PHONY: clean-all checksums
.PHONY: npm-version npm-packages npm-pack npm-publish-all

# Variables
BINARY_NAME=rg
PACKAGE_NAME=go-ripgrep
VERSION=$(shell git describe --tags --abbrev=0 2>/dev/null || echo "0.0.1")
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION)"
GOBUILD_FLAGS=-trimpath -mod=vendor
DIST_DIR=dist
CHECKSUM_FILE=$(DIST_DIR)/checksums.txt

# UPX compression
USE_UPX ?= true
ifeq ($(shell which upx 2>/dev/null),)
USE_UPX = false
endif
ifeq ($(USE_UPX),true)
UPX_CMD = upx -9
else
UPX_CMD = @true
endif

# Default target
help:
	@echo "go-ripgrep Build System"
	@echo ""
	@echo "Build targets:"
	@echo "  build            Build for current platform"
	@echo "  build-linux      Build for Linux (amd64, arm64, loong64)"
	@echo "  build-linux-loong64 Build for Linux LoongArch64"
	@echo "  build-linux-musl Build for Linux musl (amd64)"
	@echo "  build-darwin     Build for macOS (amd64, arm64)"
	@echo "  build-windows    Build for Windows (amd64, arm64)"
	@echo "  build-all        Build for all platforms and architectures"
	@echo ""
	@echo "Distribution targets:"
	@echo "  dist             Build all distribution packages"
	@echo "  dist-linux       Build Linux packages (tar.gz)"
	@echo "  dist-darwin      Build macOS packages (tar.gz)"
	@echo "  dist-windows     Build Windows packages (zip)"
	@echo "  dist-tarball     Build tarball packages only"
	@echo "  dist-zip         Build zip packages only"
	@echo ""
	@echo "NPM targets:"
	@echo "  npm-version      Sync version to npm package"
	@echo "  npm-packages     Build platform-specific npm packages"
	@echo "  npm-pack         Pack main + all platform packages"
	@echo "  npm-publish-all  Publish main + all platform packages"
	@echo ""
	@echo "Other targets:"
	@echo "  install          Install via go install"
	@echo "  test             Run tests"
	@echo "  fmt              Format code"
	@echo "  clean            Remove build artifacts"
	@echo "  clean-all        Remove everything including dist and npm artifacts"
	@echo "  checksums        Generate checksums for all dist files"
	@echo "  run              Build and run"
	@echo "  help             Show this help"

# Build for current platform
build:
	go build $(GOBUILD_FLAGS) $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/rg

# Platform builds
build-linux:
	@echo "Building for Linux..."
	@mkdir -p bin
	GOOS=linux GOARCH=amd64 go build $(GOBUILD_FLAGS) $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 ./cmd/rg
	GOOS=linux GOARCH=arm64 go build $(GOBUILD_FLAGS) $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-arm64 ./cmd/rg
	GOOS=linux GOARCH=loong64 go build $(GOBUILD_FLAGS) $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-loong64 ./cmd/rg
	@echo "Compressing Linux amd64 binary with UPX..."
	$(UPX_CMD) bin/$(BINARY_NAME)-linux-amd64

build-linux-loong64:
	@echo "Building for Linux LoongArch64..."
	@mkdir -p bin
	GOOS=linux GOARCH=loong64 go build $(GOBUILD_FLAGS) $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-loong64 ./cmd/rg

# musl: static build with CGO_ENABLED=0
build-linux-musl:
	@echo "Building for Linux musl..."
	@mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(GOBUILD_FLAGS) $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-musl-amd64 ./cmd/rg
	@echo "Compressing Linux musl binary with UPX..."
	$(UPX_CMD) bin/$(BINARY_NAME)-linux-musl-amd64

build-darwin:
	@echo "Building for macOS..."
	@mkdir -p bin
	GOOS=darwin GOARCH=amd64 go build $(GOBUILD_FLAGS) $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-amd64 ./cmd/rg
	GOOS=darwin GOARCH=arm64 go build $(GOBUILD_FLAGS) $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-arm64 ./cmd/rg

build-windows:
	@echo "Building for Windows..."
	@mkdir -p bin
	GOOS=windows GOARCH=amd64 go build $(GOBUILD_FLAGS) $(LDFLAGS) -o bin/$(BINARY_NAME)-windows-amd64.exe ./cmd/rg
	GOOS=windows GOARCH=arm64 go build $(GOBUILD_FLAGS) $(LDFLAGS) -o bin/$(BINARY_NAME)-windows-arm64.exe ./cmd/rg
	@echo "Compressing Windows amd64 binary with UPX..."
	$(UPX_CMD) bin/$(BINARY_NAME)-windows-amd64.exe

build-freebsd:
	@echo "Building for FreeBSD..."
	@mkdir -p bin
	GOOS=freebsd GOARCH=amd64 go build $(GOBUILD_FLAGS) $(LDFLAGS) -o bin/$(BINARY_NAME)-freebsd-amd64 ./cmd/rg
	GOOS=freebsd GOARCH=arm64 go build $(GOBUILD_FLAGS) $(LDFLAGS) -o bin/$(BINARY_NAME)-freebsd-arm64 ./cmd/rg

# Build all platforms
build-all: build-linux build-linux-musl build-darwin build-windows build-freebsd
	@echo ""
	@echo "Build complete! Binaries in bin/"
	@ls -lh bin/

# Install
install:
	go install -mod=vendor $(GOBUILD_FLAGS) $(LDFLAGS) ./cmd/rg

# Test
test:
	go test -v -race -mod=vendor ./...

# Format
fmt:
	gofmt -w .
	go imports -w . 2>/dev/null || go fmt ./...

# Clean
clean:
	rm -rf bin/
	rm -rf npm/packages/
	rm -rf npm/bin/
	rm -f npm/*.tgz

# Clean all
clean-all: clean
	rm -rf $(DIST_DIR)

# Run
run: build
	./bin/$(BINARY_NAME)

# Distribution: tar.gz for Linux and macOS
dist-tarball: build-linux build-linux-musl build-darwin build-freebsd
	@echo ""
	@echo "Creating tarball packages..."
	@for arch in amd64 arm64 loong64; do \
		echo "  Packaging $(PACKAGE_NAME)-linux-$${arch}.tar.gz..."; \
		./scripts/build-tarball.sh linux $${arch} $(VERSION); \
	done
	@for arch in amd64 arm64; do \
		echo "  Packaging $(PACKAGE_NAME)-darwin-$${arch}.tar.gz..."; \
		./scripts/build-tarball.sh darwin $${arch} $(VERSION); \
	done
	@for arch in amd64 arm64; do \
		echo "  Packaging $(PACKAGE_NAME)-freebsd-$${arch}.tar.gz..."; \
		./scripts/build-tarball.sh freebsd $${arch} $(VERSION); \
	done
	@echo "  Packaging $(PACKAGE_NAME)-linux-musl-amd64.tar.gz..."; \
	./scripts/build-tarball.sh linux-musl amd64 $(VERSION)

# Distribution: zip for Windows
dist-zip: build-windows
	@echo ""
	@echo "Creating Windows zip packages..."
	@for arch in amd64 arm64; do \
		echo "  Packaging $(PACKAGE_NAME)-windows-$${arch}.zip..."; \
		./scripts/build-zip.sh $${arch} $(VERSION); \
	done

# Platform distributions
dist-linux: dist-tarball
	@echo "Linux packages complete!"

dist-darwin: dist-tarball
	@echo "macOS packages complete!"

dist-windows: dist-zip
	@echo "Windows packages complete!"

# Generate checksums
checksums:
	@echo "Generating checksums..."
	@mkdir -p $(DIST_DIR)
	@cd $(DIST_DIR) && \
	find . -type f \( -name "*.tar.gz" -o -name "*.zip" \) | sort | \
	while read f; do \
		sha256sum "$$f"; \
	done > checksums.txt
	@echo "Checksums written to $(CHECKSUM_FILE)"
	@cat $(CHECKSUM_FILE)

# Build all distribution packages
dist: dist-linux dist-darwin dist-windows checksums
	@echo ""
	@echo "=========================================="
	@echo "All distribution packages built!"
	@echo ""
	@echo "Location: $(DIST_DIR)/"
	@echo ""
	@ls -lh $(DIST_DIR)/*/* 2>/dev/null || true
	@echo ""
	@echo "Checksums: $(CHECKSUM_FILE)"
	@echo "=========================================="

# NPM targets
npm-version:
	./scripts/sync-npm-version.sh $(VERSION)

npm-packages: build-all
	./scripts/build-npm-packages.sh

npm-pack: npm-version npm-packages
	@echo "Packing platform packages..."
	@for d in npm/packages/*/; do \
		if [ -f "$$d/package.json" ]; then \
			echo "  Packing $$(basename $$d)..."; \
			cd "$$d" && npm pack && cd - > /dev/null; \
			mv "$$d"/*.tgz npm/ 2>/dev/null || true; \
		fi; \
	done
	@echo "Packing main package..."
	cd npm && npm pack
	@echo "Done. Tarballs in npm/"

npm-publish-all: npm-version npm-packages
	@echo "Publishing platform packages..."
	@for d in npm/packages/*/; do \
		if [ -f "$$d/package.json" ]; then \
			echo "  Publishing $$(basename $$d)..."; \
			cd "$$d" && npm publish --tag latest && cd - > /dev/null; \
		fi; \
	done
	@echo "Publishing main package..."
	cd npm && npm publish --tag latest
	@echo "Published all packages!"
