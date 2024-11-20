#!/bin/bash
# Deploy proxies for Policy Set P1
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
  SERVICES="frontend recommendation checkout"
  pushd $TESTBED/scripts/deployment/boutique
  ./partial_istio_init.sh $SERVICES
  popd
elif [ "$SCENARIO" == "istio" ]; then
  pushd $TESTBED/scripts/deployment/boutique
  ./run_query.sh --mesh istio --init
  popd
elif [ "$SCENARIO" == "hypo" ]; then
  # Deploy Istio proxies at select services.
  SERVICES="frontend recommendation checkout"
  pushd $TESTBED/scripts/deployment/boutique
  ./partial_istio_init.sh $SERVICES
  popd
elif [ "$SCENARIO" == "devbest" ]; then
  # Deploy Istio proxies at select services.
  SERVICES="frontend recommendation checkout cart"
  pushd $TESTBED/scripts/deployment/boutique
  ./partial_istio_init.sh $SERVICES
  popd
else
  echo "Unknown scenario: $SCENARIO"
  exit 1
fi