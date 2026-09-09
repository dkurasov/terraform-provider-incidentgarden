.PHONY: build fmt test check testacc dev

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

testacc:
	TF_ACC=1 go test ./internal/provider -v -timeout 30m
