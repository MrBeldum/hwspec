VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
PREFIX  ?= $(HOME)/.local
LDFLAGS := -s -w -X main.version=$(VERSION)

# CGO off gives a fully static binary that runs on any x86_64/arm64 distro,
# including NixOS and musl-based ones.
export CGO_ENABLED = 0

.PHONY: build test release install update-ids clean

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o build/hwspec ./cmd/hwspec

test:
	go vet ./...
	go test ./...

release:
	@for arch in amd64 arm64; do \
		GOARCH=$$arch go build -trimpath -ldflags "$(LDFLAGS)" -o build/hwspec ./cmd/hwspec && \
		tar -C build -czf build/hwspec-$(VERSION)-linux-$$arch.tar.gz hwspec && \
		echo "build/hwspec-$(VERSION)-linux-$$arch.tar.gz"; \
	done
	@cd build && sha256sum hwspec-$(VERSION)-*.tar.gz > SHA256SUMS

install: build
	install -Dm755 build/hwspec $(DESTDIR)$(PREFIX)/bin/hwspec

# Refresh the embedded ID databases from upstream.
IDS_TMP := build/ids-src
update-ids:
	mkdir -p $(IDS_TMP)
	curl -fsSL -o $(IDS_TMP)/pci.ids https://pci-ids.ucw.cz/v2.2/pci.ids
	curl -fsSL -o $(IDS_TMP)/usb.ids http://www.linux-usb.org/usb.ids
	curl -fsSL -o $(IDS_TMP)/pnp.ids https://raw.githubusercontent.com/vcrhonek/hwdata/master/pnp.ids
	curl -fsSL -o $(IDS_TMP)/oui.txt https://standards-oui.ieee.org/oui/oui.txt
	curl -fsSL -o $(IDS_TMP)/decode-dimms https://git.kernel.org/pub/scm/utils/i2c-tools/i2c-tools.git/plain/eeprom/decode-dimms
	curl -fsSL -o $(IDS_TMP)/amdgpu.ids https://gitlab.freedesktop.org/mesa/drm/-/raw/main/data/amdgpu.ids
	go run ./tools/genids gzip  $(IDS_TMP)/pci.ids      internal/ids/data/pci.ids.gz
	go run ./tools/genids gzip  $(IDS_TMP)/usb.ids      internal/ids/data/usb.ids.gz
	go run ./tools/genids gzip  $(IDS_TMP)/pnp.ids      internal/ids/data/pnp.ids.gz
	go run ./tools/genids gzip  $(IDS_TMP)/amdgpu.ids   internal/ids/data/amdgpu.ids.gz
	go run ./tools/genids oui   $(IDS_TMP)/oui.txt      internal/ids/data/oui.ids.gz
	go run ./tools/genids jedec $(IDS_TMP)/decode-dimms internal/ids/data/jedec.ids.gz
	go test ./internal/ids/

clean:
	rm -rf build
