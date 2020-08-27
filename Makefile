# read PKG_VERSION from VERSION file
include VERSION

# if GOOS is not supplied, set default value based on user's system, will be overridden for OS specific packaging commands
GOOS ?= $(shell go env GOOS)
ifeq ($(GOOS),windows)
EXTENSION := .exe
endif

# if ARCH is not supplied, set default value based on user's system
ARCH ?= $(shell if [ `getconf LONG_BIT` == "64" ]; then echo "x86_64"; else echo "x86"; fi)

# set GOARCH based on ARCH
ifeq ($(ARCH),x86_64)
GOARCH := amd64
else ifeq ($(ARCH),x86)
GOARCH := 386
else
$(error "ARCH must be set to one of: x86, x86_64")
endif

# set default docker build image name
BUILD_IMAGE_NAME ?= otelopscol-build

OTELCOL_BINARY = google-cloudops-opentelemetry-collector_$(GOOS)_$(GOARCH)$(EXTENSION)

ALL_SRC := $(shell find . -name '*.go' -type f | sort)
ALL_DOC := $(shell find . \( -name "*.md" -o -name "*.yaml" \) -type f | sort)
GIT_SHA := $(shell git rev-parse --short HEAD)

BUILD_INFO_IMPORT_PATH := github.com/GoogleCloudPlatform/opentelemetry-operations-collector/internal/version
BUILD_X1 := -X $(BUILD_INFO_IMPORT_PATH).GitHash=$(GIT_SHA)
BUILD_X2 := -X $(BUILD_INFO_IMPORT_PATH).Version=$(PKG_VERSION)
LD_FLAGS := -ldflags "${BUILD_X1} ${BUILD_X2}"

.EXPORT_ALL_VARIABLES:

.DEFAULT_GOAL := presubmit

# --------------------------
#  Build / Package Commands
# --------------------------

.PHONY: install-tools
install-tools:
	go get github.com/client9/misspell/cmd/misspell
	go get github.com/golangci/golangci-lint/cmd/golangci-lint
	go get github.com/google/addlicense
	go get github.com/google/googet/goopack
	go get github.com/pavius/impi/cmd/impi

# lint / build / test

.PHONY: presubmit
presubmit: checklicense impi lint misspell test

.PHONY: checklicense
checklicense:
	@output=`addlicense -check $(ALL_SRC)` && echo checklicense finished successfully || (echo checklicense errors: $$output && exit 1)

.PHONY: impi
impi:
	@output=`impi --local github.com/GoogleCloudPlatform/opentelemetry-operations-collector --scheme stdThirdPartyLocal ./...` && echo impi finished successfully || (echo impi errors:\\n$$output && exit 1)

.PHONY: lint
lint:
	golangci-lint run

.PHONY: misspell
misspell:
	@output=`misspell -error $(ALL_DOC)` && echo misspell finished successfully || (echo misspell errors:\\n$$output && exit 1)

.PHONY: build
build:
	go build -o ./bin/$(OTELCOL_BINARY) $(LD_FLAGS) ./cmd/otelopscol

.PHONY: test
test:
	go test -v -race ./...

# googet (Windows)

.PHONY: build-goo
build-goo:
	make GOOS=windows build package-goo

.PHONY: package-goo
package-goo: export GOOS=windows
package-goo: SHELL:=/bin/bash
package-goo:
	mkdir -p dist
	# goopack doesn't support variable replacement or command line args so just use envsubst
	goopack -output_dir ./dist <(envsubst < ./.build/googet/google-cloudops-opentelemetry-collector.goospec)
	chmod -R a+rwx ./dist/

# tarball

.PHONY: build-tarball
build-tarball: build package-tarball

.PHONY: package-tarball
package-tarball:
	bash ./.build/tar/generate_tar.sh
	chmod -R a+rwx ./dist/

# --------------------
#  Create build image
# --------------------

.PHONY: docker-build-image
docker-build-image:
	docker build -t $(BUILD_IMAGE_NAME) ./.build

# -------------------------------------------
#  Run targets inside the docker build image
# -------------------------------------------

# Usage:   make TARGET=<target> docker-run
# Example: make TARGET=build-goo docker-run
.PHONY: docker-run
docker-run:
ifndef TARGET
	$(error "TARGET is undefined")
endif
	docker run -e PKG_VERSION -e ARCH -v $(CURDIR):/mnt -w /mnt $(BUILD_IMAGE_NAME) /bin/bash -c "make $(TARGET)"
