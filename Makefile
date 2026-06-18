.PHONY: all build test clean

APP     := urt
SRC     := ./...
LDFLAGS := -s -w

all: build

build:
	go build -ldflags="$(LDFLAGS)" -o $(APP) .

test:
	go test -v -count=1 $(SRC)

test-race:
	go test -v -race -count=1 $(SRC)

clean:
	rm -f $(APP)
	go clean $(SRC)
