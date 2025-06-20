#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

function usage {
    cat <<EOF
Usage: $(basename "$0") { core | ephemeral | crds | all }
Example:
   $(basename "$0") core
EOF
}

function source::settings {
    SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd -P)"
    SCRIPT_ROOT="${SCRIPT_DIR}/.."
    CODEGEN_PKG="$(go env GOMODCACHE)/$(go list -f '{{.Path}}@{{.Version}}' -m k8s.io/code-generator)"
    THIS_PKG="github.com/yaroslavborbat/sandbox-mommy/api"

    source "${CODEGEN_PKG}/kube_codegen.sh"
}

function generate::core {
    kube::codegen::gen_helpers                                    \
        --boilerplate "${SCRIPT_ROOT}/../hack/boilerplate.go.txt" \
        "${SCRIPT_ROOT}/core"

    kube::codegen::gen_client                                     \
        --with-watch                                              \
        --output-dir "${SCRIPT_ROOT}/client/generated"            \
        --output-pkg "${THIS_PKG}/client/generated"               \
        --boilerplate "${SCRIPT_ROOT}/../hack/boilerplate.go.txt" \
        "${SCRIPT_ROOT}"
}

function generate::subresources {
    kube::codegen::gen_helpers                                    \
        --boilerplate "${SCRIPT_ROOT}/../hack/boilerplate.go.txt" \
        "${SCRIPT_ROOT}/subresources"

    go tool openapi-gen                                                   \
        --output-pkg "generated"                                          \
        --output-dir "${SCRIPT_ROOT}/../internal/apiserver/api/generated" \
        --go-header-file "${SCRIPT_ROOT}/../hack/boilerplate.go.txt"      \
        -r /dev/null                                                      \
        "${SCRIPT_ROOT}/subresources/v1alpha1" "k8s.io/apimachinery/pkg/apis/meta/v1" "k8s.io/apimachinery/pkg/version"
}

function generate::crds() {
    go tool controller-gen crd:maxDescLen=0 paths="${SCRIPT_ROOT}/core/v1alpha1/..." output:crd:dir="${SCRIPT_ROOT}/../crds"
}

WHAT=$1
if [ "$#" != 1 ] || [ "${WHAT}" == "--help" ] ; then
    usage
    exit
fi

case "$WHAT" in
    core)
        source::settings
        generate::core
        ;;
    subresources)
        source::settings
        generate::subresources
        ;;
    crds)
        source::settings
        generate::crds
        ;;
    all)
        source::settings
        generate::core
        generate::crds
        generate::subresources
        ;;
    *)
        echo "Invalid argument: $WHAT"
        usage
        exit 1
        ;;
esac