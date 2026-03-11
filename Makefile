.PHONY: build build-fts5 build-bleve build-nofts test cover webtest webcover e2e clean fmt lint sql install all readme dev ubuntu-deps go-deps

BINARY_NAME=disco
BUILD_TAGS=fts5

all: fmt lint sql test build webtest webbuild readme

ubuntu-deps:
	sudo apt-get update && sudo apt-get install -y \
		ffmpeg \
		pandoc \
		groff \
		calibre \
		fonts-dejavu-core \
		sqlite3 \
		libnss3 \
		libnspr4 \
		libatk1.0-0 \
		libatk-bridge2.0-0 \
		libcups2 \
		libdrm2 \
		libxkbcommon0 \
		libxcomposite1 \
		libxdamage1 \
		libxfixes3 \
		libxrandr2 \
		libgbm1 \
		libasound2t64 \
		libpango-1.0-0 \
		libcairo2

go-deps:
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install mvdan.cc/gofumpt@latest
	go install github.com/daixiang0/gci@latest

web-install:
	npm install --prefix web

webbuild:
	npm run build --prefix web

build: webbuild
	go build -tags "$(BUILD_TAGS)" -o $(BINARY_NAME) ./cmd/disco

# Build with FTS5 support (default)
build-fts5:
	$(MAKE) BUILD_TAGS=fts5 build

# Build with Bleve full-text search support
build-bleve:
	$(MAKE) BUILD_TAGS=bleve build

# Build without any full-text search (LIKE only)
build-nofts:
	$(MAKE) BUILD_TAGS="" build

dev:
	(sleep 2 && xdg-open http://localhost:5555) &
	air -d

readme: build
	./$(BINARY_NAME) readme > README.md

test:
	go test -tags "$(BUILD_TAGS)" ./...

cover:
	go test -tags "$(BUILD_TAGS)" -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | awk '{n=split($$NF,a,"%%"); if (a[1] < 85) print $$0}' | sort -k3 -n

webtest:
	npm test --prefix web

webcover:
	npm run cover --prefix web

e2e-install:
	cd e2e && npm install && npx playwright install --with-deps

e2e-init: build
	./e2e/fixtures/init-db.sh

e2e: e2e-init
	cd e2e && npx playwright test

e2e-cli: e2e-init
	cd e2e && npx playwright test --grep 'cli-'

e2e-web: e2e-init
	cd e2e && npx playwright test --grep-invert 'cli-'

fmt:
	gofmt -s -w -e .
	go fix -tags "$(BUILD_TAGS)" ./...
	-goimports -w -e .
	-gofumpt -w .
	-gci write .

lint:
	-staticcheck -tags "$(BUILD_TAGS)" ./...
	go vet -tags "$(BUILD_TAGS)" ./...

sql:
	sqlc generate
	sqlc vet
	-sqlc diff

clean:
	rm -f $(BINARY_NAME)
	rm -f test.db
	rm -f coverage.out

# Install the binary to $GOPATH/bin
install:
	go install -tags "$(BUILD_TAGS)" ./cmd/disco

release-build: webbuild
	mkdir -p dist
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -tags "$(BUILD_TAGS)" -o dist/$(BINARY_NAME)-$(GOOS)-$(GOARCH) ./cmd/disco
