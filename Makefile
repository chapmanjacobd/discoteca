.PHONY: build test cover webtest webcover e2e e2e-ui e2e-debug e2e-install clean fmt lint sql install all readme dev

BINARY_NAME=disco
BUILD_TAGS=fts5

all: fmt lint sql test webtest build readme

build:
	go build -tags "$(BUILD_TAGS)" -o $(BINARY_NAME) ./cmd/disco

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

# E2E Tests (requires built binary)
e2e-install:
	cd e2e && npm install && npx playwright install chromium

e2e: build e2e-install
	cd e2e && npx playwright test --project=chromium

e2e-ui: build e2e-install
	cd e2e && npx playwright test --ui

e2e-debug: build e2e-install
	cd e2e && npx playwright test --debug

e2e-headed: build e2e-install
	cd e2e && npx playwright test --headed

e2e-report:
	cd e2e && npx playwright show-report

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
