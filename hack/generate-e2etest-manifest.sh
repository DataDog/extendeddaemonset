#!/usr/bin/env bash

set -e

SCRIPT_DIR=$(dirname "${BASH_SOURCE:-0}")
MANIFEST_FILE_PATH="$SCRIPT_DIR/../test/e2e/global-manifest.yaml"

cat /dev/null > $MANIFEST_FILE_PATH
for crd in $(find $SCRIPT_DIR/../deploy/crds/v1beta1 -type f -iname '*.yaml')
do
  echo "---" >> $MANIFEST_FILE_PATH
  cat $crd >> $MANIFEST_FILE_PATH
done