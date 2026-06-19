.PHONY: all build test clean fmt

APP     := urt
SRC     := ./...
LDFLAGS := -s -w

all: fmt build

build:
	go build -ldflags="$(LDFLAGS)" -o $(APP) .

test: fmt
	go test -v -count=1 $(SRC)

test-race: fmt
	go test -v -race -count=1 $(SRC)

fmt:
	go fmt $(SRC)

clean:
	rm -f $(APP)
	go clean $(SRC)
