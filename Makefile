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
		cp LICENSE dist/LICENSE && tar -czf dist/$(BINARY_NAME)-darwin-amd64.tar.gz -C dist $(BINARY_NAME) LICENSE && rm dist/$(BINARY_NAME) dist/LICENSE
	GOOS=darwin  GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY_NAME) . && \
		cp LICENSE dist/LICENSE && tar -czf dist/$(BINARY_NAME)-darwin-arm64.tar.gz -C dist $(BINARY_NAME) LICENSE && rm dist/$(BINARY_NAME) dist/LICENSE
	GOOS=linux   GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY_NAME) . && \
		cp LICENSE dist/LICENSE && tar -czf dist/$(BINARY_NAME)-linux-amd64.tar.gz -C dist $(BINARY_NAME) LICENSE && rm dist/$(BINARY_NAME) dist/LICENSE
	GOOS=linux   GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY_NAME) . && \
		cp LICENSE dist/LICENSE && tar -czf dist/$(BINARY_NAME)-linux-arm64.tar.gz -C dist $(BINARY_NAME) LICENSE && rm dist/$(BINARY_NAME) dist/LICENSE
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY_NAME).exe . && \
		cp LICENSE dist/LICENSE && tar -czf dist/$(BINARY_NAME)-windows-amd64.tar.gz -C dist $(BINARY_NAME).exe LICENSE && rm dist/$(BINARY_NAME).exe dist/LICENSE


