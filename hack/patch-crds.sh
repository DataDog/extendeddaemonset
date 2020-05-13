#!/usr/bin/env bash

set -e

SCRIPT_DIR=$(dirname "${BASH_SOURCE:-0}")
YQ="$SCRIPT_DIR/../bin/yq"

# Update the `protocol` attribute of v1.ContainerPort to required as default is not yet supported
# See: https://github.com/kubernetes/api/blob/master/core/v1/types.go#L2165
# Until issue is fixed: https://github.com/kubernetes-sigs/controller-tools/issues/438 and integrated in operator-sdk
$YQ m -i "$SCRIPT_DIR/../deploy/crds/datadoghq.com_extendeddaemonsetreplicasets_crd.yaml" "$SCRIPT_DIR/patch-crd-protocol-kube1.18.yaml"
$YQ m -i "$SCRIPT_DIR/../deploy/crds/datadoghq.com_extendeddaemonsets_crd.yaml" "$SCRIPT_DIR/patch-crd-protocol-kube1.18.yaml"
