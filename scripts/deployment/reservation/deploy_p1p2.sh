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
  SERVICES="frontend search"
  pushd $TESTBED/scripts/deployment/reservation
  ./partial_istio_init.sh $SERVICES
  popd

  # Add Cilium annotations for the rest of the services
  CILIUM_SERVICES="profile recommendation reservation user geo rate"
  for SERVICE in $CILIUM_SERVICES; do
      kubectl annotate service $SERVICE service.cilium.io/lb-l7=enabled --overwrite
  done
elif [ "$SCENARIO" == "istio" ]; then
  pushd $TESTBED/scripts/deployment/reservation
  ./run_query.sh --mesh istio --init
  popd
elif [ "$SCENARIO" == "hypo" ]; then
  # Deploy Istio proxies at select services.
  SERVICES="frontend search geo profile rate recommendation reservation user"
  pushd $TESTBED/scripts/deployment/reservation
  ./partial_istio_init.sh $SERVICES
  popd
else
  echo "Unknown scenario: $SCENARIO"
  exit 1
fi