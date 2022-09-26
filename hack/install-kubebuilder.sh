#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

ROOT=$(git rev-parse --show-toplevel)
WORK_DIR=$(mktemp -d)
cleanup() {
  rm -rf "$WORK_DIR"
}
trap "cleanup" EXIT SIGINT

VERSION=$1

if [ -z "$VERSION" ];
then
  echo "usage: bin/install-kubebuilder.sh <version>"
  exit 1
fi

# download kubebuilder and extract it to tmp
rm -rf "$ROOT/bin/kubebuilder"
curl -L -o kubebuilder https://go.kubebuilder.io/dl/latest/$(go env GOOS)/$(go env GOARCH) > "$ROOT/bin/kubebuilder"
