.PHONY: all build clean deb install test

BINARY_NAME := router-manager-batch
VERSION := $(shell grep '^Version:' debian/control | cut -d' ' -f2 || echo "0.1.0")
ARCH := arm64

all: build

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=$(ARCH) go build -o $(BINARY_NAME) cmd/batch/main.go

clean:
	rm -f $(BINARY_NAME)
	rm -f ../router-manager-batch_*.deb
	rm -f ../router-manager-batch_*.changes
	rm -f ../router-manager-batch_*.buildinfo
	rm -rf debian/router-manager-batch/
	rm -rf debian/.debhelper/
	rm -f debian/debhelper-build-stamp
	rm -f debian/files
	rm -f debian/*.debhelper
	rm -f debian/*.substvars

deb: clean
	# Build the binary first
	$(MAKE) build
	# Copy binary to debian directory for packaging
	cp $(BINARY_NAME) debian/
	# Build the Debian package
	dpkg-buildpackage -b -rfakeroot -us -uc --host-arch=$(ARCH)
	# Move the built package to current directory
	mv ../router-manager-batch_*.deb ./
	@echo "Package built: router-manager-batch_$(VERSION)_$(ARCH).deb"

install:
	install -D -m 0755 $(BINARY_NAME) $(DESTDIR)/usr/local/bin/$(BINARY_NAME)
	install -D -m 0644 debian/router-manager-batch.service $(DESTDIR)/lib/systemd/system/router-manager-batch.service
	install -D -m 0644 debian/router-manager-batch.timer $(DESTDIR)/lib/systemd/system/router-manager-batch.timer
	install -D -m 0600 debian/router-manager-batch.default $(DESTDIR)/etc/default/router-manager-batch

test:
	go test -v ./...

dev-run:
	go run cmd/batch/

# Helper targets for development
.PHONY: lint fmt

lint:
	golangci-lint run ./...

fmt:
	gofumpt -w .
	golangci-lint run --fix
