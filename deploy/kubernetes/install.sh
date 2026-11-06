#!/bin/bash

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)

kubectl create configmap examplesrv-config --from-file "${SCRIPT_DIR}/conf.toml" -n iam
kubectl create -f "${SCRIPT_DIR}/deployment.yaml" -n iam
kubectl create -f "${SCRIPT_DIR}/service.yaml" -n iam
