#!/bin/bash
# Deploy proxies for Policy Set P1
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
  pushd $TESTBED/scripts/deployment/social
  ./run_query.sh --mesh cilium --init
  popd
  # Iterate over all services and deploy proxies
  SERVICES="url-shorten-service text-service user-mention-service home-timeline-service compose-post-service user-service social-graph-service user-timeline-service post-storage-service"
  for SERVICE in $SERVICES; do
      kubectl annotate service $SERVICE service.cilium.io/lb-l7=enabled --overwrite
  done
elif [ "$SCENARIO" == "istio" ]; then
  pushd $TESTBED/scripts/deployment/social
  ./run_query.sh --mesh istio --init
  popd
elif [ "$SCENARIO" == "hypothetical" ]; then
  # Deploy Istio proxies at select services.
  SERVICES="text-service home-timeline-service compose-post-service user-service social-graph-service user-timeline-service url-shorten-service user-mention-service post-storage-service"
  pushd $TESTBED/scripts/deployment/social
  ./partial_istio_init.sh $SERVICES
  popd
elif [ "$SCENARIO" == "devbest" ]; then
  ISTIO_SERVICES="text-service home-timeline-service compose-post-service user-service social-graph-service user-timeline-service"
  pushd $TESTBED/scripts/deployment/social
  ./partial_istio_init.sh $ISTIO_SERVICES
  popd

  # Add Cilium annotations for the rest of the services
  CILIUM_SERVICES="url-shorten-service user-mention-service post-storage-service"
  for SERVICE in $CILIUM_SERVICES; do
      kubectl annotate service $SERVICE service.cilium.io/lb-l7=enabled --overwrite
  done
else
  echo "Unknown scenario: $SCENARIO"
  exit 1
fi