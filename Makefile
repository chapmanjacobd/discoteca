.PHONY: build build-fts5 test cover webtest webcover e2e clean fmt lint install all readme dev ubuntu-deps go-deps web-install webbuild e2e-install e2e-init e2e-cli e2e-web release-build benchmark benchstat profiles screenshots astiav-build astiav-test astiav-shell

BINARY_NAME=disco
BUILD_TAGS=fts5

ifeq ($(OS),Windows_NT)
	EXE=.exe
else
	EXE=
endif

all: clean webbuild fmt lint build test webtest readme

ubuntu-deps:
	sudo apt-get update && sudo apt-get install -y --no-install-recommends -o APT::Install-Suggests=0 \
		fonts-dejavu-core \
		libasound2t64 \
		libatk-bridge2.0-0 \
		libatk1.0-0 \
		libcairo2 \
		libcups2 \
		libxcb-cursor0 \
		libdbus-1-3 \
		libdrm2 \
		libegl1 \
		libexpat1 \
		libfontconfig1 \
		libfreetype6 \
		libgbm1 \
		libglib2.0-0 \
		libnspr4 \
		libnss3 \
		libopengl0 \
		libpango-1.0-0 \
		libx11-6 \
		libxcb1 \
		libxcomposite1 \
		libxdamage1 \
		libxfixes3 \
		libxkbcommon0 \
		libxrandr2 \
		ghostscript \
		sqlite3 \
		ffmpeg \
		groff \
		pandoc \
		wget
	sudo -v && wget -qO- https://download.calibre-ebook.com/linux-installer.sh | sudo sh /dev/stdin

macos-deps:
	-brew install --formula ffmpeg pandoc sqlite || true
	-brew install --cask calibre

go-deps:
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install mvdan.cc/gofumpt@latest
	go install github.com/daixiang0/gci@latest
	go install gotest.tools/gotestsum@latest

deps-update:
	go get -u ./...
	go mod tidy

web-install:
	cd web && npm install

webbuild:
	cd web && npm run build

build: webbuild
	go build -tags "$(BUILD_TAGS)" -o $(BINARY_NAME)$(EXE) ./cmd/disco

# Build with FTS5 support (default)
build-fts5:
	$(MAKE) BUILD_TAGS=fts5 build

dev: build
	go run -tags fts5 ./cmd/disco serve -v --dev --public-dir web/dist audio.db books.db images.db video.db

test:
	gotestsum --format pkgname-and-test-fails -- ./... -tags "$(BUILD_TAGS)"

cover:
	gotestsum --format pkgname-and-test-fails -- ./... -tags "$(BUILD_TAGS)" -coverprofile=coverage.out
	go tool cover -func=coverage.out | awk '{n=split($$NF,a,"%%"); if (a[1] < 85) print $$0}' | sort -k3 -n

webtest:
	cd web && npm test

webcover:
	cd web && npm run cover

fmt:
	gofmt -s -w -e .
	-goimports -w -e .
	-gofumpt -w .
	-gci write .
	go fix -tags "$(BUILD_TAGS)" ./...

lint:
	-staticcheck -tags "$(BUILD_TAGS)" ./...
	go vet -tags "$(BUILD_TAGS)" ./...

install:
	go install -tags "$(BUILD_TAGS)" ./cmd/disco

e2e-install:
	npm install --prefix e2e
	npm run install --prefix e2e

e2e-init: build
	./e2e/fixtures/init-db.sh $(BINARY_NAME)$(EXE)

e2e: e2e-init
	npm run test --prefix e2e

e2e-cli: e2e-init
	npm run test --prefix e2e -- --grep 'cli-'

e2e-web: e2e-init
	npm run test --prefix e2e -- --grep-invert 'cli-'

release-build: webbuild
	mkdir -p dist
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -tags "$(BUILD_TAGS)" -o dist/$(BINARY_NAME)-$(GOOS)-$(GOARCH)$(EXE) ./cmd/disco

benchmark:
	go test -tags "$(BUILD_TAGS)" -bench=. -benchmem -benchtime=2s ./...

benchmark-save:
	go test -tags "$(BUILD_TAGS)" -bench=. -benchmem -benchtime=5s -count=5 ./... > benchmark-$(shell date +%Y%m%d-%H%M%S).txt

benchstat:
	@echo "Usage: make benchstat old=old-benchmarks.txt new=new-benchmarks.txt"
	benchstat $(old) $(new)

profiles:
	go test -tags "$(BUILD_TAGS)" -bench=BenchmarkSearch -benchtime=10s -cpuprofile=cpu.prof -memprofile=mem.prof -trace=trace.out -o commands.test ./internal/commands/
	@echo "Profiles generated: cpu.prof, mem.prof, trace.out"
	@echo "View with: go tool pprof -http=:8080 commands.test cpu.prof"

profiles-svg: profiles
	go tool pprof -svg -output=cpu-profile.svg commands.test cpu.prof
	go tool pprof -svg -output=mem-profile.svg commands.test mem.prof
	@echo "SVG profiles generated: cpu-profile.svg, mem-profile.svg"

screenshots: build
	cd e2e && npx playwright test -c playwright.screenshots.config.ts

# ==============================================================================
# Astiav Backend (FFmpeg 8.0 via libavformat CGO)
# Uses Podman container with RPM Fusion for FFmpeg 8.0 dev libraries
# ==============================================================================

# Build astiav container image (Fedora Rawhide + RPM Fusion + FFmpeg 8.0)
astiav-image:
	@echo "Building astiav container image with FFmpeg 8.0..."
	podman build -f Containerfile.astiav -t disco-astiav:latest .
	@echo "Image built: disco-astiav:latest"

# Build with astiav backend using Podman container
astiav-build: astiav-image
	@echo "Building with astiav backend in Podman container..."
	podman run --rm --security-opt label=disable \
		-v $(PWD):/src:z \
		-w /src \
		disco-astiav:latest \
		sh -c '\
			ffmpeg -version | head -1 && \
			CGO_CFLAGS="-I/usr/include" CGO_LDFLAGS="-L/usr/lib64" \
				go build -tags "fts5 astiav" -o /src/$(BINARY_NAME)-astiav /src/cmd/disco && \
			echo "Built: $(BINARY_NAME)-astiav"'
	@echo "Binary created: $(BINARY_NAME)-astiav"

# Build with astiav backend using static linking (larger binary, no dependencies)
# Requires FFmpeg built with --enable-static --disable-shared
astiav-build-static: astiav-image
	@echo "Building with astiav backend (static linking)..."
	@echo "Note: This creates a large (~100MB) standalone binary"
	podman run --rm --security-opt label=disable \
		-v $(PWD):/src:z \
		-w /src \
		disco-astiav:latest \
		sh -c '\
			/usr/local/bin/ffmpeg -version | head -1 && \
			CGO_CFLAGS="-I/usr/local/include" \
			CGO_LDFLAGS="-L/usr/local/lib -static" \
			go build -tags "fts5 astiav" \
				-ldflags "-extldflags '-static' -s -w" \
				-o /src/$(BINARY_NAME)-astiav-static /src/cmd/disco && \
			echo "Built: $(BINARY_NAME)-astiav-static" && \
			ls -lh /src/$(BINARY_NAME)-astiav-static && \
			file /src/$(BINARY_NAME)-astiav-static'
	@echo "Static binary created: $(BINARY_NAME)-astiav-static"
	@echo "Test with: podman run --rm -v $(PWD):/src:z -w /src disco-astiav:latest ./$(BINARY_NAME)-astiav-static --help"

# Test with astiav backend using Podman container
astiav-test: astiav-image
	@echo "Running tests with astiav backend in Podman container..."
	podman run --rm --security-opt label=disable \
		-v $(PWD):/src:z \
		-w /src \
		disco-astiav:latest \
		sh -c '\
			ffmpeg -version | head -1 && \
			CGO_CFLAGS="-I/usr/include" CGO_LDFLAGS="-L/usr/lib64" \
				go test -tags "fts5 astiav" -v ./internal/metadata/... -run TestFFProbeBackend'
	@echo "Astiav tests completed"

# Interactive shell in container with FFmpeg 8.0 dev libraries (for debugging)
astiav-shell: astiav-image
	@echo "Starting interactive shell in Podman container with FFmpeg 8.0..."
	podman run -it --rm --security-opt label=disable \
		-v $(PWD):/src:z \
		-w /src \
		disco-astiav:latest \
		bash

# Quick test: build and run astiav backend test in one command (builds image first)
astiav-quicktest: astiav-image
	@echo "Quick astiav backend test..."
	podman run --rm --security-opt label=disable \
		-v $(PWD):/src:z \
		-w /src \
		disco-astiav:latest \
		sh -c '\
			CGO_CFLAGS="-I/usr/include" CGO_LDFLAGS="-L/usr/lib64" \
				go test -tags "fts5 astiav" -run TestFFProbeBackend_RealMedia ./internal/metadata/ -v'

# Native build with astiav (requires FFmpeg 8.0 dev libs installed on host)
astiav-build-native:
	@echo "Building with astiav backend (native, requires FFmpeg 8.0 dev libs)..."
	@ffmpeg_version=$$(ffmpeg -version 2>/dev/null | head -1 | grep -oP '\d+\.\d+' | head -1) && \
	if [ "$$ffmpeg_version" != "8.0" ]; then \
		echo "Warning: FFmpeg version $$ffmpeg_version detected, astiav requires 8.0"; \
		echo "Use 'make astiav-build' for containerized build with FFmpeg 8.0"; \
	fi
	CGO_CFLAGS="-I/usr/include" CGO_LDFLAGS="-L/usr/lib64" \
		go build -tags "fts5 astiav" -o $(BINARY_NAME)-astiav$(EXE) ./cmd/disco
	@echo "Binary created: $(BINARY_NAME)-astiav$(EXE)"

readme: build
	./$(BINARY_NAME)$(EXE) readme > README.md

clean:
	rm -f $(BINARY_NAME)
	rm -f test.db
	rm -f $(BINARY_NAME).exe
	rm -f coverage.out
	rm -f *.prof *.out *.svg
	rm -f benchmark-*.txt
	rm -rf web/dist/*
