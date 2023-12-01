#!/usr/bin/env bash
# Run query for the Hotel Reservation benchmark with Istio service mesh
# Arguments:
# 1: First time install, Init (1) else (0)

if [ $# -lt 1 ]; then
  echo 1>&2 "Not enough arguments"
  exit 2
fi

: "${TESTBED:=$HOME}"
pushd $TESTBED

if [[ "$1" -eq "1" ]]; then
  sudo apt install -y luarocks
  sudo luarocks install luasocket

  # Pull docker image 
  docker pull divyanshus/hotelreservation
  
  git clone https://github.com/delimitrou/DeathStarBench.git
  pushd DeathStarBench/hotelReservation
  kubectl apply -Rf kubernetes/
  popd

  # Make wrk2 executable
  pushd DeathStarBench/wrk2
  make -j 4
  popd
  
  pushd $TESTBED/istio-1.16.1
  # Get Istio inject configurations
  kubectl -n istio-system get configmap istio-sidecar-injector -o=jsonpath='{.data.config}' > inject-config.yaml
  kubectl -n istio-system get configmap istio-sidecar-injector -o=jsonpath='{.data.values}' > inject-values.yaml
  kubectl -n istio-system get configmap istio -o=jsonpath='{.data.mesh}' > mesh-config.yaml
  
  services=("frontend" "reserve" "user")
  for service in "${services[@]}"
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

  # Use the Istio Ingress Gateway
  pushd scripts
  kubectl apply -f deployment/hotel-reservation/reservation-gateway.yaml
  popd

  # Wait for the pods to get running
  sleep 3m
fi

INGRESS_HOST=$(kubectl get po -l istio=ingressgateway -n istio-system -o jsonpath='{.items[0].status.hostIP}')
INGRESS_PORT=$(kubectl -n istio-system get service istio-ingressgateway -o jsonpath='{.spec.ports[?(@.name=="http2")].nodePort}')

GATEWAY_URL=$INGRESS_HOST:$INGRESS_PORT

# Warm-up
pushd DeathStarBench/hotelReservation
../wrk2/wrk -D exp -t 5 -c 10 -d 10 -L -s ./wrk2/scripts/hotel-reservation/mixed-workload_type_1.lua http://$GATEWAY_URL -R 100

# Measure CPU and Memory
sudo python3 $TESTBED/scripts/utils/cpumem_stats.py reservation_partialistio > $TESTBED/logs/python.log 2>&1 &

# Run queries to log timings
../wrk2/wrk -D exp -t 10 -c 10 -d 40 -L -s ./wrk2/scripts/hotel-reservation/mixed-workload_type_1.lua http://$GATEWAY_URL -R 50 >> $TESTBED/out/time_reservation_partialistio.run 2>&1

# Kill CPU and Memory measurement
pid=$(pgrep -f cpumem)
sudo kill -SIGINT $pid

popd