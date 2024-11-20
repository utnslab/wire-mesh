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
  # pushd $TESTBED/scripts/deployment/social
  # ./run_query.sh --mesh cilium --init
  # popd

  SERVICES="nginx-thrift text-service home-timeline-service compose-post-service social-graph-service"
  pushd $TESTBED/scripts/deployment/social
  ./partial_istio_init.sh $SERVICES
  popd

  # Iterate over all services and deploy proxies
  SERVICES="text-service home-timeline-service compose-post-service social-graph-service user-timeline-service"
  for SERVICE in $SERVICES; do
      kubectl annotate service $SERVICE service.cilium.io/lb-l7=enabled --overwrite
  done
elif [ "$SCENARIO" == "istio" ]; then
  pushd $TESTBED/scripts/deployment/social
  ./run_query.sh --mesh istio --init
  popd
elif [ "$SCENARIO" == "hypo" ]; then
  # Deploy Istio proxies at select services.
  SERVICES="nginx-thrift text-service home-timeline-service compose-post-service user-service social-graph-service user-timeline-service url-shorten-service user-mention-service post-storage-service"
  pushd $TESTBED/scripts/deployment/social
  ./partial_istio_init.sh $SERVICES
  popd
else
  echo "Unknown scenario: $SCENARIO"
  exit 1
fi