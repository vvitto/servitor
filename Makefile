BINARY := servitor

.PHONY: build test lint run clean

build:
	go build -o bin/$(BINARY) ./cmd/servitor

lint:
	gofmt -l . && go vet ./...

run: build
	./bin/$(BINARY)

clean:
	rm -rf bin
