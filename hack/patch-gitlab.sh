#!/bin/bash

set -e

SCRIPTS_DIR="$(dirname "$0")"
source "$SCRIPTS_DIR/os-env.sh"

ROOT_DIR=$(git rev-parse --show-toplevel)
GITLAB_FILE_PATH=$ROOT_DIR/.gitlab-ci.yml
SERVICE_FILE_PATH=$ROOT_DIR/service.datadog.yaml

# Read arg as VERSION
VERSION=$1

if [ -z "$VERSION" ];
then
  echo "usage: hack/patch-gitlab.sh <version>"
  exit 1
fi

# Update CONDUCTOR_BUILD_TAG envvar in Gitlab file, which is used in Conductor-triggered image build jobs
# Replace . with -, add v
VERSION_SLUG=v$(echo $VERSION | sed "s/\./-/g")
# Update envvar in Gitlab file
$SED "s/CONDUCTOR_BUILD_TAG: .*$/CONDUCTOR_BUILD_TAG: $VERSION_SLUG/g" $GITLAB_FILE_PATH

# Update branch that runs Conductor job, add v
BRANCH=v$(echo $VERSION | sed -r "s/([^.]+.[^.]*).*/\1/")
$SED "s/branch: .*$/branch: \"$BRANCH\"/g" $SERVICE_FILE_PATH
