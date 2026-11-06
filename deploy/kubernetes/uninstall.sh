#!/bin/bash

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)

kubectl delete -f "${SCRIPT_DIR}/service.yaml" -n iam
kubectl delete -f "${SCRIPT_DIR}/deployment.yaml" -n iam
kubectl delete configmap examplesrv-config  -n iam
