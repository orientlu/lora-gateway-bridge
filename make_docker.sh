#!/bin/bash
# by orientlu

image="gatewayserver"
ver="t.0.0.1"
prefix="ccr.ccs.tencentyun.com/lora/"

docker build -t ${image}:${ver} .
docker tag "${image}:${ver}" "${prefix}${image}:${ver}"
