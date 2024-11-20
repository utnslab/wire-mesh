#!/bin/bash
# Deploy proxies for Policy Set P2
# Args:
#  $1: Scenario (istio/hypo/devbest/wire)

: "${TESTBED:=$HOME}"

if [ "$#" -ne 1 ]; then
  echo "Usage: $0 <scenario>"
  exit 1
fi

SCENARIO=$1

# Check if SCENARIO is wire
if [ "$SCENARIO" == "wire" ]; then
  # pushd $TESTBED/scripts/deployment/boutique
  # ./run_query.sh --mesh cilium --init
  # popd

  SERVICES="frontend recommendation checkout"
  pushd $TESTBED/scripts/deployment/boutique
  ./partial_istio_init.sh $SERVICES
  popd

  # Add Cilium annotations for the cart service
  kubectl annotate service cartservice service.cilium.io/lb-l7=enabled --overwrite
elif [ "$SCENARIO" == "istio" ]; then
  pushd $TESTBED/scripts/deployment/boutique
  ./run_query.sh --mesh istio --init
  popd
elif [ "$SCENARIO" == "hypo" ]; then
  SERVICES="frontend recommendation checkout cart"
  pushd $TESTBED/scripts/deployment/boutique
  ./partial_istio_init.sh $SERVICES
  popd
else
  echo "Unknown scenario: $SCENARIO"
  exit 1
fi