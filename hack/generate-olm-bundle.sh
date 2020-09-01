#!/usr/bin/env bash
set -euo pipefail

# Use GNU tools, even on MacOS
if sed --version 2>/dev/null | grep -q "GNU sed"; then
    SED=sed
elif gsed --version 2>/dev/null | grep -q "GNU sed"; then
    SED=gsed
fi

ROOT=$(git rev-parse --show-toplevel)
OLM_FOLDER=$ROOT/deploy/olm-catalog/extendeddaemonset
IMAGE_NAME='datadog/extendeddaemonset'
REDHAT_REGISTRY='registry.connect.redhat.com/'
REDHAT_IMAGE_NAME="${REDHAT_REGISTRY}${IMAGE_NAME}"
ZIP_FILE_NAME=$ROOT/dist/olm-redhat-bundle.zip

WORK_DIR=$(mktemp -d)
trap 'rm -rf "$WORK_DIR"' EXIT

# move all zip file if exit
if [[ -e "$ZIP_FILE_NAME" ]]; then
    mv "$ZIP_FILE_NAME" "$ZIP_FILE_NAME.old"
fi

for i in "$OLM_FOLDER"/$VERSION/*.yaml "$OLM_FOLDER"/*.yaml; do
    $SED "s|${IMAGE_NAME}|${REDHAT_IMAGE_NAME}|g" < "$i" > "$WORK_DIR/${i##*/}"
done

cd "$WORK_DIR"
$SED -e 's/packageName\: extendeddaemonset/packageName\: extendeddaemonset-certified/g' datadog-operator.package.yaml
rm -- *.bak
zip "$ZIP_FILE_NAME" -- *.yaml
