#!/usr/bin/env bash
# Initialize reservation application for the Wire service mesh with Istio proxies for only some services.

: "${TESTBED:=$HOME}"

KUBE_SERVICES=("user" "search" "consul" "jaeger" "geo" "profile" "rate" "reccomend" "reserve")
ISTIO_SERVICES=("frontend")

pushd $TESTBED/DeathStarBench/hotelReservation/kubernetes
for service in "${KUBE_SERVICES[@]}"; do
  kubectl apply -Rf $service/
done
popd

pushd $TESTBED/istio-1.16.1
# Get Istio inject configurations
kubectl -n istio-system get configmap istio-sidecar-injector -o=jsonpath='{.data.config}' > inject-config.yaml
kubectl -n istio-system get configmap istio-sidecar-injector -o=jsonpath='{.data.values}' > inject-values.yaml
kubectl -n istio-system get configmap istio -o=jsonpath='{.data.mesh}' > mesh-config.yaml

for service in "${ISTIO_SERVICES[@]}"
do
  ./bin/istioctl kube-inject \
  --injectConfigFile inject-config.yaml \
  --meshConfigFile mesh-config.yaml \
  --valuesFile inject-values.yaml \
  --filename $TESTBED/DeathStarBench/hotelReservation/kubernetes/$service/$service-service.yaml \
  | kubectl apply -f -

  ./bin/istioctl kube-inject \
  --injectConfigFile inject-config.yaml \
  --meshConfigFile mesh-config.yaml \
  --valuesFile inject-values.yaml \
  --filename $TESTBED/DeathStarBench/hotelReservation/kubernetes/$service/$service-deployment.yaml \
  | kubectl apply -f -
done
popd
