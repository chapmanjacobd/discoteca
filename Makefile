.PHONY: build build-fts5 test cover webtest webcover e2e clean fmt lint install all readme dev ubuntu-deps go-deps web-install webbuild e2e-install e2e-init e2e-cli e2e-web release-build benchmark benchstat profiles screenshots

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
