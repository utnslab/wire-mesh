#!/bin/bash
# Deploy proxies for Policy Set P2
# Args:
#  $1: Scenario (istio/hypothetical/devbest/wire)

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

  # # Iterate over all services and deploy proxies
  # SERVICES="frontend recommendation checkout cart"
  # for SERVICE in $SERVICES; do
  #   name=$SERVICE
  #   if [ "$SERVICE" == "frontend" ]; then
  #     name="frontend"
  #   else
  #     name=$SERVICE"service"
  #   fi
  #   kubectl annotate service $name service.cilium.io/lb-l7=enabled --overwrite
  # done

  SERVICES="frontend recommendation checkout"
  pushd $TESTBED/scripts/deployment/boutique
  ./partial_istio_init.sh $SERVICES
  popd
elif [ "$SCENARIO" == "istio" ]; then
  pushd $TESTBED/scripts/deployment/boutique
  ./run_query.sh --mesh istio --init
  popd
elif [ "$SCENARIO" == "hypothetical" ]; then
  SERVICES="frontend recommendation checkout cart"
  pushd $TESTBED/scripts/deployment/boutique
  ./partial_istio_init.sh $SERVICES
  popd
elif [ "$SCENARIO" == "devbest" ]; then
  ISTIO_SERVICES="frontend recommendation"
  pushd $TESTBED/scripts/deployment/boutique
  ./partial_istio_init.sh $ISTIO_SERVICES
  popd

  # Add Cilium annotations for the rest of the services
  CILIUM_SERVICES="checkout"
  for SERVICE in $CILIUM_SERVICES; do
    kubectl annotate service ${SERVICE}service service.cilium.io/lb-l7=enabled --overwrite
  done
else
  echo "Unknown scenario: $SCENARIO"
  exit 1
fi