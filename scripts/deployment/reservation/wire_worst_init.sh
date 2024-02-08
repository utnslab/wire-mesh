#!/usr/bin/env bash
# Initialize reservation application for the Wire service mesh with Istio proxies for only some services.

: "${TESTBED:=$HOME}"

KUBE_SERVICES=("consul" "jaeger")
ISTIO_SERVICES=("frontend" "user" "search" "geo" "profile" "rate" "reccomend" "reserve")

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
  name=$service
  if [[ $service == "reccomend" ]]; then
    name="recommendation"
  elif [[ $service == "reserve" ]]; then
    name="reservation"
  fi

  for file in $TESTBED/DeathStarBench/hotelReservation/kubernetes/$service/*; do
    filename=$(basename "$file")

    # Use Istio for only service and deployment files, use kubectl for the rest
    if [[ $filename == $name"-service.yaml" || $filename == $name"-deployment.yaml" ]]; then
      ./bin/istioctl kube-inject \
      --injectConfigFile inject-config.yaml \
      --meshConfigFile mesh-config.yaml \
      --valuesFile inject-values.yaml \
      --filename $file \
      | kubectl apply -f -
    else
      kubectl apply -f $file
    fi
  done
done
popd

# Wait for 1 minute and remove pods with (Evicted|Unknown|Completed|Error) status
sleep 1m
kubectl get pods | grep Evicted | awk '{print $1}' | xargs kubectl delete pod
kubectl get pods | grep Unknown | awk '{print $1}' | xargs kubectl delete pod
kubectl get pods | grep Completed | awk '{print $1}' | xargs kubectl delete pod
kubectl get pods | grep Error | awk '{print $1}' | xargs kubectl delete pod

# Wait for another 1 minute
sleep 1m
