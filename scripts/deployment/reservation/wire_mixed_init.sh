#!/usr/bin/env bash
# Initialize reservation application for the Wire service mesh with linkerd and Istio proxies.

: "${TESTBED:=$HOME}"

# Services to use the Wire service mesh.
LINKERD_SERVICES=("frontend" "search" "consul" "jaeger")
ISTIO_SERVICES=("user" "geo" "profile" "rate" "reccomend" "reserve")

pushd $TESTBED/DeathStarBench/hotelReservation/kubernetes
for service in "${LINKERD_SERVICES[@]}"; do
  $TESTBED/.linkerd2/bin/linkerd inject $service/ | kubectl apply -f -
done
popd

# Use wire with istio injection only where needed
pushd $TESTBED/istio-1.16.1

# Get Istio inject configurations
kubectl -n istio-system get configmap istio-sidecar-injector -o=jsonpath='{.data.config}' > inject-config.yaml
kubectl -n istio-system get configmap istio-sidecar-injector -o=jsonpath='{.data.values}' > inject-values.yaml
kubectl -n istio-system get configmap istio -o=jsonpath='{.data.mesh}' > mesh-config.yaml

for service in "${ISTIO_SERVICES[@]}"; do
  for file in $TESTBED/DeathStarBench/hotelReservation/kubernetes/$service/*; do
    ./bin/istioctl kube-inject \
    --injectConfigFile inject-config.yaml \
    --meshConfigFile mesh-config.yaml \
    --valuesFile inject-values.yaml \
    --filename $file \
    | kubectl apply -f -
  done
done
popd
