PREV_VERSION=$(shell git describe --abbrev=0 --tags)
VERSION=$(shell echo $$(($(PREV_VERSION)+1)))
LDFLAGS := -X github.com/jhrv/testapp/pkg/version.Revision=$(shell git rev-parse --short HEAD) -X github.com/jhrv/testapp/pkg/version.Version=$(VERSION)

release:
	go build -a -installsuffix cgo -o testapp -ldflags "-s $(LDFLAGS)"
