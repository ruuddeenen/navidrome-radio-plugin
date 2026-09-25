# Build tooling for the Navidrome Radio Plugin.
#
# The Navidrome plugin PDK is not published as a versioned Go module, so `deps`
# clones Navidrome at the pinned version into .build/navidrome and go.mod points
# its `replace` directive there.

NAVIDROME_VERSION ?= v0.64.1
NAVIDROME_SRC     ?= $(CURDIR)/.build/navidrome
PLUGIN_ID         := navidrome-radio-plugin
GO_IMAGE          ?= golang:1.25
TINYGO_IMAGE      ?= tinygo/tinygo:latest

.PHONY: all manifest deps fmt vet test test-docker build build-go build-wasm package build-docker build-go-docker clean

all: build

manifest:
	python3 scripts/gen_manifest.py

deps:
	mkdir -p .build
	test -d $(NAVIDROME_SRC)/plugins/pdk/go || \
		git clone --depth 1 --branch $(NAVIDROME_VERSION) https://github.com/navidrome/navidrome.git $(NAVIDROME_SRC)

fmt:
	gofmt -w main.go internal

vet: deps
	go vet ./...

test: deps
	go test ./internal/...

test-docker: deps
	docker run --rm -v $(CURDIR):/app -v /tmp/go-cache:/root/.cache/go-build -v /tmp/go-mod:/go/pkg/mod \
		-w /app $(GO_IMAGE) go test ./internal/...

# Build the WASM module. Requires TinyGo installed locally.
build-wasm: deps
	mkdir -p .build
	tinygo build -o .build/plugin.wasm -target wasip1 -buildmode=c-shared .

# Package manifest.json + plugin.wasm into a .ndp (zip) file.
package: manifest
	cd .build && rm -f ../$(PLUGIN_ID).ndp && \
		( zip -j ../$(PLUGIN_ID).ndp manifest.json plugin.wasm || \
		  python3 -c "import zipfile;z=zipfile.ZipFile('../$(PLUGIN_ID).ndp','w');z.write('manifest.json');z.write('plugin.wasm');z.close()" )

build: build-wasm package

# Standard Go build (Go 1.24+ wasip1, no TinyGo). Much lighter than TinyGo.
build-go: deps manifest
	mkdir -p .build
	GOOS=wasip1 GOARCH=wasm go build -buildvcs=false -buildmode=c-shared -o .build/plugin.wasm .
	$(MAKE) package

# Build inside the TinyGo container (used on hosts without Go/TinyGo).
build-docker: deps manifest
	mkdir -p .build
	docker run --rm -v $(CURDIR):/app -v /tmp/go-cache:/root/.cache/go-build -v /tmp/go-mod:/go/pkg/mod \
		-w /app $(TINYGO_IMAGE) tinygo build -o .build/plugin.wasm -target wasip1 -buildmode=c-shared .
	$(MAKE) package

# Standard Go build inside a container (recommended on small hosts).
build-go-docker: deps manifest
	mkdir -p .build
	docker run --rm --cpus=2 --memory=2g -e GOOS=wasip1 -e GOARCH=wasm \
		-v $(CURDIR):/app -v /tmp/go-cache:/root/.cache/go-build -v /tmp/go-mod:/go/pkg/mod \
		-w /app $(GO_IMAGE) go build -buildvcs=false -buildmode=c-shared -o .build/plugin.wasm .
	sh -c 'cd .build && rm -f ../$(PLUGIN_ID).ndp && python3 -c "import zipfile;z=zipfile.ZipFile(\"../$(PLUGIN_ID).ndp\",\"w\");z.write(\"manifest.json\");z.write(\"plugin.wasm\");z.close()"'

# Run unit tests inside a container (hosts without Go).
test-go-docker: deps
	docker run --rm --cpus=2 --memory=2g -v $(CURDIR):/app \
		-v /tmp/go-cache:/root/.cache/go-build -v /tmp/go-mod:/go/pkg/mod \
		-w /app $(GO_IMAGE) go test ./internal/...

clean:
	rm -rf .build $(PLUGIN_ID).ndp
