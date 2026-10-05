VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
PREFIX  ?= $(HOME)/.local
LDFLAGS := -s -w -X main.version=$(VERSION)

# CGO off gives a fully static binary that runs on any x86_64/arm64 distro,
# including NixOS and musl-based ones.
export CGO_ENABLED = 0

.PHONY: build test release install fetch-ids gen-ids update-ids clean

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

# ID databases. fetch-ids downloads upstream sources; gen-ids converts them
# into DIR with a manifest (PREV keeps dates for unchanged undated files).
IDS_SRC := build/ids-src
DIR     ?= internal/ids/data
PREV    ?= $(DIR)/manifest.json

fetch-ids:
	mkdir -p $(IDS_SRC)
	curl -fsSL --retry 3 -o $(IDS_SRC)/pci.ids https://pci-ids.ucw.cz/v2.2/pci.ids
	curl -fsSL --retry 3 -o $(IDS_SRC)/usb.ids http://www.linux-usb.org/usb.ids
	curl -fsSL --retry 3 -o $(IDS_SRC)/pnp.ids https://raw.githubusercontent.com/vcrhonek/hwdata/master/pnp.ids
	curl -fsSL --retry 3 -o $(IDS_SRC)/oui.txt https://standards-oui.ieee.org/oui/oui.txt
	curl -fsSL --retry 3 -o $(IDS_SRC)/decode-dimms https://git.kernel.org/pub/scm/utils/i2c-tools/i2c-tools.git/plain/eeprom/decode-dimms
	curl -fsSL --retry 3 -o $(IDS_SRC)/amdgpu.ids https://gitlab.freedesktop.org/mesa/drm/-/raw/main/data/amdgpu.ids

gen-ids:
	mkdir -p $(DIR)
	go run ./tools/genids gzip  $(IDS_SRC)/pci.ids      $(DIR)/pci.ids.gz
	go run ./tools/genids gzip  $(IDS_SRC)/usb.ids      $(DIR)/usb.ids.gz
	go run ./tools/genids gzip  $(IDS_SRC)/pnp.ids      $(DIR)/pnp.ids.gz
	go run ./tools/genids gzip  $(IDS_SRC)/amdgpu.ids   $(DIR)/amdgpu.ids.gz
	go run ./tools/genids oui   $(IDS_SRC)/oui.txt      $(DIR)/oui.ids.gz
	go run ./tools/genids jedec $(IDS_SRC)/decode-dimms $(DIR)/jedec.ids.gz
	go run ./tools/genids manifest $(DIR) $(PREV)

# Refresh the copies embedded in the binary.
update-ids: fetch-ids gen-ids
	go test ./internal/ids/

clean:
	rm -rf build
