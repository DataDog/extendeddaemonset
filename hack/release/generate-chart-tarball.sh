#!/bin/bash
set -e

source "$(dirname $0)/../os-env.sh"

TAG=""
if [ $# -gt 0 ]; then
    TAG=$1
    echo "TAG=$TAG"
else
    echo "First parameter should be the new TAG"
    exit 1
fi
VERSION=${TAG:1}

GIT_ROOT=$(git rev-parse --show-toplevel)
OUTPUT_FOLDER=$GIT_ROOT/dist

cp -Lr $GIT_ROOT/chart/* $OUTPUT_FOLDER/

for CHART in extendeddaemonset
do
    find $OUTPUT_FOLDER/$CHART -name Chart.yaml | xargs $SED "s/PLACEHOLDER_VERSION/$VERSION/g"
    find $OUTPUT_FOLDER/$CHART -name values.yaml | xargs $SED "s/PLACEHOLDER_VERSION/$VERSION/g"
    tar -zcvf $OUTPUT_FOLDER/$CHART.tar.gz -C $OUTPUT_FOLDER $CHART
done
