#!/usr/bin/env bash

uname_os() {
  os=$(uname -s | tr '[:upper:]' '[:lower:]')
  case "$os" in
    msys_nt) os="windows" ;;
  esac
  echo "$os"
}

OS=$(uname_os)
ARCH=$(uname -m)
PLATFORM="$OS-$ARCH"
ROOT=$(git rev-parse --show-toplevel)
export SED="sed -i"
if [ "$PLATFORM" == "darwin-arm64" ]; then
    export SED="sed -i ''"
elif [ "$OS" == "darwin" ]; then
    export SED="sed -i .bak"
fi
