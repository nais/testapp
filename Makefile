LDFLAGS := -X github.com/jhrv/testapp/version.Revision=$(shell git rev-parse --short HEAD) -X github.com/jhrv/testapp/version.Version=$(shell /bin/cat ./version)

release:
	go build -a -installsuffix cgo -o testapp -ldflags "-s $(LDFLAGS)"
