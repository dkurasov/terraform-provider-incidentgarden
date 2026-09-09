.PHONY: build fmt test check testacc dev changelog-new changelog-check changelog-release

CHANGIE ?= changie

build:
	go build ./...

dev:
	go build -o ./bin/terraform-provider-incidentgarden .

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

test:
	go test ./...

check:
	go mod tidy
	git diff --exit-code -- go.mod go.sum
	go vet ./...
	golangci-lint run
	go test ./...
	terraform fmt -check -recursive examples
	$(MAKE) changelog-check

changelog-new:
	$(CHANGIE) new

changelog-check:
	$(CHANGIE) batch 0.1.0 --dry-run >/dev/null

changelog-release:
	@test -n "$(VERSION)" || (echo "VERSION is required (for example: make changelog-release VERSION=0.1.0)"; exit 1)
	$(CHANGIE) batch "$(VERSION)"
	$(CHANGIE) merge

testacc:
	TF_ACC=1 go test ./internal/provider -v -timeout 30m
