default: build

build:
	go build -o terraform-provider-devin

install: build
	mkdir -p ~/.terraform.d/plugins/registry.terraform.io/cognition-ai/devin/0.1.0/linux_amd64/
	cp terraform-provider-devin ~/.terraform.d/plugins/registry.terraform.io/cognition-ai/devin/0.1.0/linux_amd64/

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
	rm -f terraform-provider-devin

.PHONY: build install test testintegration testacc lint fmt clean
