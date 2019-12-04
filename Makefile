DATE=$(shell date "+%Y-%m-%d")
LAST_COMMIT=$(shell git --no-pager log -1 --pretty=%h)
VERSION="$(DATE)-$(LAST_COMMIT)"
LDFLAGS := -X github.com/nais/testapp/pkg/version.Revision=$(shell git rev-parse --short HEAD) -X github.com/nais/testapp/pkg/version.Version=$(VERSION)

all:
	go build -o bin/testapp cmd/testapp/main.go

release:
	go build -a -installsuffix cgo -o bin/testapp -ldflags "-s $(LDFLAGS)" cmd/testapp/main.go

local:
	go run cmd/testapp/main.go --bind-address=127.0.0.1:8080
