#!/usr/bin/env bash

set -e
set -o pipefail

# Find all packages containing Go test files. If a package does not contain test files
# (or the test files are excluded because of their build tags), it will not be
# included in this list.
packages=$(go list -f '{{.ImportPath}}: {{.XTestGoFiles}} {{.TestGoFiles}}' ./... | grep -v '\[\] \[\]' | cut -f 1 -d ":")

install_etcd() {
  ETCD_VERSION="3.3.10"
  echo "Installing etcd $ETCD_VERSION..."
  wget -q -O /tmp/etcd.tar.gz "https://github.com/etcd-io/etcd/releases/download/v$ETCD_VERSION/etcd-v$ETCD_VERSION-linux-amd64.tar.gz"
  tar xvzf /tmp/etcd.tar.gz -C /tmp
  export PATH="$PATH:/tmp/etcd-v$ETCD_VERSION-linux-amd64"
  echo
}

# Only install the etcd command if we test the etcd3locker package
(echo "$packages" | grep etcd3locker > /dev/null) && install_etcd

while read package; do
  dir="${package/'github.com/tus/tusd'/.}"
  echo "Testing package $package ($dir):"
  go get -v -t "$dir"
  go test -v "$dir"
  go vet "$dir"
  echo
done <<< "$packages"
