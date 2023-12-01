#!/bin/bash
# Arguments:
# 1: Name of the mesh to use ('I' for Istio, 'L' for Linkerd, 'P' for Plain, 'N' for NGINX)

if [ $# -lt 1 ]; then
  echo 1>&2 "Not enough arguments"
  exit 2
fi

# Setup service mesh for the given argument
: "${TESTBED:=$HOME}"
pushd $TESTBED/scripts

# Install service mesh
./mesh_setup.sh $1

if [[ "$1" == "I" ]]; then
  # Run the experiment for Istio
  ./deployment/hotel-reservation/run_query_istio.sh 1
elif [[ "$1" == "L" ]]; then
  # Run the experiment for Linkerd
  ./deployment/hotel-reservation/run_query_linkerd.sh 1
elif [[ "$1" == "P" ]]; then
  # Run the experiment for Plain
  ./deployment/hotel-reservation/run_query_plain.sh 1
elif [[ "$1" == "N" ]]; then
  # Run the experiment for NGINX
  ./deployment/hotel-reservation/run_query_nginx.sh 1
fi

# Uninstall service mesh
./mesh_uninstall.sh $1
