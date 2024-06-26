#!/bin/bash
# Deploy proxies for Policy Set P1
# Args:
#  $1: Scenario (motivation 1, 2, 3)

: "${TESTBED:=$HOME}"

if [ "$#" -ne 1 ]; then
  echo "Usage: $0 <scenario>"
  exit 1
fi

SCENARIO=$1

# Check the SCENARIO 
if [ "$SCENARIO" == "1" ]; then
  # Deploy Istio proxies at select services.
  SERVICES="frontend"
  pushd $TESTBED/scripts/deployment/reservation
  ./partial_istio_init.sh $SERVICES
  popd
elif [ "$SCENARIO" == "2" ]; then
  # Deploy Istio proxies at select services.
  SERVICES="frontend search"
  pushd $TESTBED/scripts/deployment/reservation
  ./partial_istio_init.sh $SERVICES
  popd
elif [ "$SCENARIO" == "3" ]; then
  # Deploy Istio proxies at select services.
  SERVICES="frontend search geo rate"
  pushd $TESTBED/scripts/deployment/reservation
  ./partial_istio_init.sh $SERVICES
  popd
else
  echo "Unknown scenario: $SCENARIO"
  exit 1
fi