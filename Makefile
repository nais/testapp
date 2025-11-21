.PHONY: all build release local test check fmt

all:
	mise run all

build:
	mise run build

local:
	mise run local

test:
	mise run test

check:
	mise run check

staticcheck:
	mise run check:staticcheck

vulncheck:
	mise run check:vulncheck

fmt:
	mise run fmt
