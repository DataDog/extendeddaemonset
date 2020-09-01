#!/usr/bin/env bash
set -euo pipefail

# Locate project root
ROOT=$(git rev-parse --show-toplevel)

pushd $ROOT/deploy/olm-catalog/extendeddaemonset/
PREVIOUS_VERSION=$(ls -dUt */ | grep -v \'-rc\' | head -1 | sed 's/.$//')
echo "PREVIOUS_VERSION=$PREVIOUS_VERSION"
popd

VERSION=""
RELEASE_CANDIDATE=""
if [ $# -gt 0 ]; then
    VERSION=$1
    echo "VERSION=$VERSION"
else
    echo "First parameter should be the new VERSION"
    exit 1
fi

if [ $# -gt 1 ]; then
    RELEASE_CANDIDATE=$2
    echo "RELEASE_CANDIDATE=$RELEASE_CANDIDATE"
    VERSION=$VERSION"-rc."$RELEASE_CANDIDATE
fi

VVERSION="v$VERSION"
pushd $ROOT

# Use GNU tools, even on MacOS
if sed --version 2>/dev/null | grep -q "GNU sed"; then
    SED=sed
elif gsed --version 2>/dev/null | grep -q "GNU sed"; then
    SED=gsed
fi


# Update chart version
"$ROOT/bin/yq" w -i "$ROOT/chart/extendeddaemonset/Chart.yaml" "appVersion" "$VVERSION"
"$ROOT/bin/yq" w -i "$ROOT/chart/extendeddaemonset/Chart.yaml" "version" "$VVERSION"
"$ROOT/bin/yq" w -i "$ROOT/chart/extendeddaemonset/values.yaml" "image.tag" "$VVERSION"

# Upadte version in deploy folder
"$ROOT/bin/yq" w -i "$ROOT/deploy/operator.yaml" "spec.template.spec.containers[0].image" "datadog/extendeddaemonset:$VVERSION"

# Run OLM generation
make VERSION=$VERSION generate-olm

# Patch OLM Generation
OLM_FILE=$ROOT/deploy/olm-catalog/extendeddaemonset/$VERSION/extendeddaemonset.clusterserviceversion.yaml
$ROOT/bin/yq m -i --overwrite --autocreate=true $OLM_FILE $ROOT/hack/release/cluster-service-version-patch.yaml
$ROOT/bin/yq w -i $OLM_FILE "spec.customresourcedefinitions.owned[0].displayName" "ExtendedDaemonset"
$ROOT/bin/yq w -i $OLM_FILE "spec.replaces" "extendeddaemonset.v$PREVIOUS_VERSION"
$ROOT/bin/yq w -i $OLM_FILE "metadata.annotations.createdAt" "$(date '+%Y-0%m-%d %T')"

# update extendeddaemonset.package.yaml
if [ -z "$RELEASE_CANDIDATE" ]; then
    # Update official channel
    $ROOT/bin/yq w -i $ROOT/deploy/olm-catalog/extendeddaemonset/extendeddaemonset.package.yaml "channels[0].currentCSV" "extendeddaemonset.$VVERSION"
fi
# always update the test channel
$ROOT/bin/yq w -i $ROOT/deploy/olm-catalog/extendeddaemonset/extendeddaemonset.package.yaml "channels[1].currentCSV" "extendeddaemonset.$VVERSION"


# cleanup tmp files
find . -name "*.bak" -type f -delete

# leave the ROOT folder
popd
