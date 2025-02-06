#!/bin/bash

set -e

SCRIPTS_DIR="$(dirname "$0")"
source "$SCRIPTS_DIR/os-env.sh"

ROOT_DIR=$(git rev-parse --show-toplevel)
FILE_PATH=$ROOT_DIR/.gitlab-ci.yml 

# read arg as VERSION
VERSION=$1

if [ -z "$VERSION" ];
then
  echo "usage: hack/patch-gitlab.sh <version>"
  exit 1
fi

# Replace . with -, add v
VERSION=v$(echo $VERSION | sed "s/\./-/g")
# Update envvar in Gitlab file
$SED "s/CONDUCTOR_BUILD_TAG: .*$/CONDUCTOR_BUILD_TAG: $VERSION/g" $FILE_PATH
