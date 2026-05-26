PROVIDER_NAME   = terraform-provider-devin
VERSION        ?= 0.1.0
DIST_DIR        = dist

PLATFORMS = \
	linux/amd64 \
	linux/arm64 \
	darwin/amd64 \
	darwin/arm64 \
	windows/amd64

default: build

build:
	go build -trimpath -ldflags="-s -w -X main.version=$(VERSION)" -o $(PROVIDER_NAME)

install: build
	mkdir -p ~/.terraform.d/plugins/registry.terraform.io/cognition-ai/devin/$(VERSION)/linux_amd64/
	cp $(PROVIDER_NAME) ~/.terraform.d/plugins/registry.terraform.io/cognition-ai/devin/$(VERSION)/linux_amd64/

# Cross-compile for all platforms (Option B)
build-all:
	@rm -rf $(DIST_DIR) && mkdir -p $(DIST_DIR)
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*}; \
		GOARCH=$${platform#*/}; \
		output="$(DIST_DIR)/$(PROVIDER_NAME)_$(VERSION)"; \
		if [ "$$GOOS" = "windows" ]; then output="$${output}.exe"; fi; \
		echo "Building $$GOOS/$$GOARCH..."; \
		CGO_ENABLED=0 GOOS=$$GOOS GOARCH=$$GOARCH \
			go build -trimpath -ldflags="-s -w -X main.version=$(VERSION)" \
			-o "$$output" .; \
		archive="$(DIST_DIR)/$(PROVIDER_NAME)_$(VERSION)_$${GOOS}_$${GOARCH}.zip"; \
		(cd $(DIST_DIR) && zip "$$(basename $$archive)" "$$(basename $$output)" && rm "$$(basename $$output)"); \
	done

checksums:
	@cd $(DIST_DIR) && shasum -a 256 *.zip > $(PROVIDER_NAME)_$(VERSION)_SHA256SUMS

manifest:
	@echo '{"version":1,"metadata":{"protocol_versions":["6.0"]}}' > $(DIST_DIR)/terraform-registry-manifest.json

test:
	go test ./internal/client/... -v -race -count=1

testintegration:
	go test ./internal/provider/... -run TestContract -v -count=1 -timeout 10m

testacc:
	TF_ACC=1 go test ./internal/provider/... -v -count=1 -timeout 30m

lint:
	go vet ./...

fmt:
	gofmt -s -w .

clean:
	rm -f $(PROVIDER_NAME)
	rm -rf $(DIST_DIR)

.PHONY: default build install build-all checksums manifest test testintegration testacc lint fmt clean
