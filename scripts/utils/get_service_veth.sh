#!/bin/bash
# Get interfaces of pods serving the given kubectl service
# Arguments:
# 1: Service Name

if [ $# -lt 1 ]; then
  echo 1>&2 "Not enough arguments"
  exit 2
fi

PODS=$(sudo crictl -r unix:///var/run/containerd/containerd.sock pods --name=$1 -q)

for POD in $PODS; do
  NETNS_PATH=$(sudo crictl -r unix:///var/run/containerd/containerd.sock inspectp $POD | jq -r '.info.runtimeSpec.linux.namespaces[] |select(.type=="network") | .path')
  NETNS=${NETNS_PATH##*/}
  INT=$(ip addr | grep -B 1 $NETNS | head -n 1 | awk '{print $2}' | sed 's/://')
  echo $INT
done