DATE=$(shell date "+%Y-%m-%d")
LAST_COMMIT=$(shell git --no-pager log -1 --pretty=%h)
VERSION="$(DATE)-$(LAST_COMMIT)"
LDFLAGS := -X github.com/nais/testapp/pkg/version.Revision=$(shell git rev-parse --short HEAD) -X github.com/nais/testapp/pkg/version.Version=$(VERSION)

release:
	go build -a -installsuffix cgo -o testapp -ldflags "-s $(LDFLAGS)"

local:
	go run *.go --bind-address=127.0.0.1:8080 --bucket-name=testapp_ci-gcp --service-account-credentials-file=./serviceaccount.json
