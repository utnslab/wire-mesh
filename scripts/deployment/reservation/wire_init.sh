#!/usr/bin/env bash
# Initialize reservation application for the Wire service mesh.

: "${TESTBED:=$HOME}"

# Services to use the Wire service mesh.
WIRE_SERVICES=("user" "geo" "profile" "rate" "reccomend" "reserve")
KUBE_SERVICES=("frontend" "search" "consul" "jaeger")

# Use Wire configs or kubernetes configs, depending on the service.
pushd $TESTBED/DeathStarBench/hotelReservation/wire
for service in "${WIRE_SERVICES[@]}"; do
  kubectl apply -Rf $service/
done
popd

pushd $TESTBED/DeathStarBench/hotelReservation/kubernetes
for service in "${KUBE_SERVICES[@]}"; do
  # kubectl apply -Rf $service/
  $TESTBED/.linkerd2/bin/linkerd inject $service/ | kubectl apply -f -
done
popd

# First start all kubernetes services
pushd $TESTBED/DeathStarBench/hotelReservation
kubectl apply -Rf kubernetes/
popd

# Use wire with istio injection only where needed
pushd $TESTBED/istio-1.16.1

# Get Istio inject configurations
kubectl -n istio-system get configmap istio-sidecar-injector -o=jsonpath='{.data.config}' > inject-config.yaml
kubectl -n istio-system get configmap istio-sidecar-injector -o=jsonpath='{.data.values}' > inject-values.yaml
kubectl -n istio-system get configmap istio -o=jsonpath='{.data.mesh}' > mesh-config.yaml

for service in "${WIRE_SERVICES[@]}"; do
  name=$service
  if [[ $service == "reccomend" ]]; then
    name="recommendation"
  elif [[ $service == "reserve" ]]; then
    name="reservation"
  fi

  ./bin/istioctl kube-inject \
  --injectConfigFile inject-config.yaml \
  --meshConfigFile mesh-config.yaml \
  --valuesFile inject-values.yaml \
  --filename $TESTBED/DeathStarBench/hotelReservation/kubernetes/$service/$name-service.yaml \
  | kubectl apply -f -

  ./bin/istioctl kube-inject \
  --injectConfigFile inject-config.yaml \
  --meshConfigFile mesh-config.yaml \
  --valuesFile inject-values.yaml \
  --filename $TESTBED/DeathStarBench/hotelReservation/kubernetes/$service/$name-deployment.yaml \
  | kubectl apply -f -
done
popd
