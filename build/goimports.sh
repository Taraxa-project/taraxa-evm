#!/bin/sh

find_files() {
  find . ! \( \
      \( \
        -path './build/_workspace' \
        -o -path './build/bin' \
        -o -path './crypto/bn256' \
        -o -path '*/vendor/*' \
      \) -prune \
    \) -name '*.go'
}

go get golang.org/x/tools/cmd/goimports

GOFMT="gofmt -s -w"
GOIMPORTS="goimports -w"
find_files | xargs ${GOFMT}
find_files | xargs ${GOIMPORTS}
