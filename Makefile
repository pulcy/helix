PROJECT := helix
SCRIPTDIR := $(shell pwd)
ROOTDIR := $(shell cd $(SCRIPTDIR) && pwd)
VERSION:= $(shell cat $(ROOTDIR)/VERSION)
COMMIT := $(shell git rev-parse --short HEAD)

GOBUILDDIR := $(SCRIPTDIR)/.gobuild
SRCDIR := $(SCRIPTDIR)
BINDIR := $(ROOTDIR)
VENDORDIR := $(ROOTDIR)/deps

ORGPATH := github.com/pulcy
ORGDIR := $(GOBUILDDIR)/src/$(ORGPATH)
REPONAME := $(PROJECT)
REPODIR := $(ORGDIR)/$(REPONAME)
REPOPATH := $(ORGPATH)/$(REPONAME)
BIN := $(BINDIR)/$(PROJECT)

GOPATH := $(GOBUILDDIR)
GOVERSION := 1.10.0-alpine
CACHEVOL := $(PROJECT)-gocache

ifndef GOOS
	GOOS := $(shell go env GOOS)
endif
ifndef GOARCH
	GOARCH := $(shell go env GOARCH)
endif

SOURCES := $(shell find $(SRCDIR) -name '*.go')


.PHONY: all
all: $(BIN)

.PHONY: clean
clean:
	rm -Rf $(BIN) $(GOBUILDDIR)

deps:
	@${MAKE} -B -s $(GOBUILDDIR)

$(GOBUILDDIR):
	@mkdir -p $(ORGDIR)
	@rm -f $(REPODIR) && ln -s ../../../.. $(REPODIR)
	GOPATH=$(GOBUILDDIR) pulsar go flatten -V $(VENDORDIR)

update-vendor:
	@rm -Rf $(VENDORDIR)
	@pulsar go vendor -V $(VENDORDIR) \
		github.com/arangodb-helper/go-certificates \
		github.com/dchest/uniuri \
		github.com/pkg/errors \
		github.com/rs/zerolog \
		github.com/spf13/cobra \
		github.com/spf13/pflag \
		gopkg.in/yaml.v2 \
		golang.org/x/sync/errgroup \
		golang.org/x/crypto/ssh

$(CACHEVOL):
	@docker volume create $(CACHEVOL)

$(BIN): $(GOBUILDDIR) $(CACHEVOL) $(SOURCES)
	docker run \
		--rm \
		-v $(ROOTDIR):/usr/code \
		-v $(CACHEVOL):/usr/gocache \
		-e GOCACHE=/usr/gocache \
		-e GOPATH=/usr/code/.gobuild \
		-e GOOS=$(GOOS) \
		-e GOARCH=$(GOARCH) \
		-e CGO_ENABLED=0 \
		-w /usr/code/ \
		golang:$(GOVERSION) \
		go build -installsuffix netgo -tags netgo -ldflags "-X main.projectVersion=$(VERSION) -X main.projectBuild=$(COMMIT)" -o /usr/code/$(PROJECT) $(REPOPATH)

