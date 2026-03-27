BINARY_NAME := kubectl-node_pods
PLUGIN_NAME := node-pods
REPO := noahfan/kubectl-node-pods
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "v0.1.0")
LDFLAGS := -s -w

.PHONY: build clean install test release test-release

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) .

install: build
	cp $(BINARY_NAME) $(shell go env GOPATH)/bin/

clean:
	rm -f $(BINARY_NAME)
	rm -rf dist/

test:
	go test ./...

release: clean
	@mkdir -p dist
	GOOS=darwin  GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY_NAME) . && \
		tar -czf dist/$(BINARY_NAME)-darwin-amd64.tar.gz -C dist $(BINARY_NAME) && rm dist/$(BINARY_NAME)
	GOOS=darwin  GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY_NAME) . && \
		tar -czf dist/$(BINARY_NAME)-darwin-arm64.tar.gz -C dist $(BINARY_NAME) && rm dist/$(BINARY_NAME)
	GOOS=linux   GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY_NAME) . && \
		tar -czf dist/$(BINARY_NAME)-linux-amd64.tar.gz -C dist $(BINARY_NAME) && rm dist/$(BINARY_NAME)
	GOOS=linux   GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY_NAME) . && \
		tar -czf dist/$(BINARY_NAME)-linux-arm64.tar.gz -C dist $(BINARY_NAME) && rm dist/$(BINARY_NAME)
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY_NAME).exe . && \
		tar -czf dist/$(BINARY_NAME)-windows-amd64.tar.gz -C dist $(BINARY_NAME).exe && rm dist/$(BINARY_NAME).exe

test-release: clean
	@set -euo pipefail; \
	OS="$$(uname | tr '[:upper:]' '[:lower:]')"; \
	ARCH_RAW="$$(uname -m)"; \
	case "$$ARCH_RAW" in \
		x86_64) ARCH="amd64" ;; \
		arm64|aarch64) ARCH="arm64" ;; \
		*) echo "Unsupported arch: $$ARCH_RAW"; exit 1 ;; \
	esac; \
	mkdir -p dist; \
	GOOS="$$OS" GOARCH="$$ARCH" go build -ldflags "$(LDFLAGS)" -o "dist/$(BINARY_NAME)" .; \
	TARBALL="dist/$(BINARY_NAME)-$$OS-$$ARCH.tar.gz"; \
	tar -czf "$$TARBALL" -C dist "$(BINARY_NAME)"; \
	SHA="$$(shasum -a 256 "$$TARBALL" | awk '{print $$1}')"; \
	PLATFORMS_JSON="$$(printf '[{"os":"%s","arch":"%s","bin":"%s","uri":"https://example.com/%s-%s-%s.tar.gz","sha256":"%s"}]' "$$OS" "$$ARCH" "$(BINARY_NAME)" "$(BINARY_NAME)" "$$OS" "$$ARCH" "$$SHA")"; \
	go run ./cmd/gen-manifest --template templates/krew-plugin.yaml.tmpl --output node-pods.yaml --plugin-name $(PLUGIN_NAME) --version "$(VERSION)" --repo $(REPO) --platforms-json "$$PLATFORMS_JSON"; \
	kubectl krew uninstall $(PLUGIN_NAME) >/dev/null 2>&1 || true; \
	kubectl krew install --manifest=./node-pods.yaml --archive="./$$TARBALL"; \
	echo "Installed $(PLUGIN_NAME) from $$TARBALL"

