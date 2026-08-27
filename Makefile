.PHONY: all build test clean fmt install

APP     := urt
SRC     := ./...
LDFLAGS := -s -w
PREFIX  ?= /usr
DESTDIR ?=

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

install: build
	install -Dm755 $(APP) $(DESTDIR)$(PREFIX)/bin/$(APP)
	install -Dm755 openrc/urt $(DESTDIR)/etc/init.d/urt
	install -Dm644 openrc/conf.d/urt $(DESTDIR)/etc/conf.d/urt
	install -dm755 $(DESTDIR)/etc/urt
	@if [ ! -f $(DESTDIR)/etc/urt/config.yaml ]; then \
		install -Dm600 config.example.yaml $(DESTDIR)/etc/urt/config.yaml; \
	fi
