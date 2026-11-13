.PHONY: build run test test-race lint fmt vet clean install snapshot
build:
	go build -trimpath -o bin/zet ./cmd/zet
run:
	go run ./cmd/zet
test:
	go test ./...
test-race:
	go test -race ./...
lint:
	golangci-lint run ./...
fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')
vet:
	go vet ./...
clean:
	rm -rf bin
install:
	go install ./cmd/zet
snapshot:
	goreleaser release --snapshot --clean
