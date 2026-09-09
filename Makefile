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
	go vet ./...
	go test ./...

testacc:
	TF_ACC=1 go test ./internal/provider -v -timeout 30m
