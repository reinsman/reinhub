#!/bin/bash

# 可从任意工作目录执行：./deploy/image/build-image.sh -p -v 2.1.0-static

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
PROJECT_ROOT=$(CDPATH= cd -- "${SCRIPT_DIR}/../.." && pwd)
BIN_DIR="${PROJECT_ROOT}/bin"
KUBERNETES_DIR="${PROJECT_ROOT}/deploy/kubernetes"

push=0
semver=""

while getopts "pv:" opt; do
  case $opt in
    p)
      push=1
      ;;
    v)
      semver=$OPTARG
      ;;
    :)
      echo "Option -$OPTARG requires an argument."
      exit 1
      ;;
    ?) #当有不认识的选项的时候arg为?
      echo "Invalid option: -$OPTARG index:$OPTIND"
      exit 2
      ;;
  esac
done

echo "push:" $push
echo "semver:" $semver


cd "${BIN_DIR}" || exit 1

APP_NAME=examplesrv
APP_VER=`git rev-list -1 HEAD | cut -c 1-8`
REGISTRY=docker.example.com/iam

# 配置文件一般和应用版本对应，打包镜像后一般在 k8s 上部署或更新。
cp -f conf.toml "${KUBERNETES_DIR}/"
docker build --build-arg "TAG=${APP_VER}" -t ${REGISTRY}/${APP_NAME}:v${APP_VER} -f "${SCRIPT_DIR}/Dockerfile" .
docker tag ${REGISTRY}/${APP_NAME}:v${APP_VER} ${REGISTRY}/${APP_NAME}:latest

if [ -n "$semver" ]; then
  docker tag ${REGISTRY}/${APP_NAME}:v${APP_VER} ${REGISTRY}/${APP_NAME}:v${semver}
fi
# docker push ${REGISTRY}/${APP_NAME}:v${APP_VER}
if [ $push == 1 ]; then
  echo "push " ${REGISTRY}/${APP_NAME}:v${APP_VER}
  docker push ${REGISTRY}/${APP_NAME}:v${APP_VER}
  echo "push " ${REGISTRY}/${APP_NAME}:latest
  docker push ${REGISTRY}/${APP_NAME}:latest
  if [ -n "$semver" ]; then
    echo "push " ${REGISTRY}/${APP_NAME}:v${semver}
    docker push ${REGISTRY}/${APP_NAME}:v${semver}
  fi
fi
