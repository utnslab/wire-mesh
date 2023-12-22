#!/usr/bin/env bash
# Run query for the Social Network benchmark with different service meshes
# Arguments:
# 1: Name of the service mesh (istio or plain or nginx)
# 2: First time install, Init (1) else (0)

if [ $# -lt 2 ]; then
  echo 1>&2 "Not enough arguments"
  exit 2
fi

: "${TESTBED:=$HOME}"
pushd $TESTBED

if [[ "$2" -eq "1" ]]; then
  sudo apt install -y luarocks
  sudo luarocks install luasocket
  
  git clone https://github.com/DivyanshuSaxena/DeathStarBench.git
  pushd DeathStarBench/socialNetwork/helm-chart
  helm install social-network ./socialnetwork
  popd

  # Make wrk2 executable
  pushd DeathStarBench/wrk2
  make -j 4
  popd

  # Wait for the pods to get running
  sleep 1m
fi

# Get the Cluster IP of the nginx web server.
SVC_HOST=$(kubectl get svc nginx-thrift -o jsonpath='{.spec.clusterIP}')
SVC_PORT=$(kubectl get svc nginx-thrift -o jsonpath='{.spec.ports[?(@.protocol=="TCP")].port}')
SVC_URL=$SVC_HOST:$SVC_PORT

if [[ "$2" -eq "1" ]]; then
  # Initialize graph
  pushd DeathStarBench/socialNetwork
  python3 scripts/init_social_graph.py --ip=$SVC_HOST --port $SVC_PORT --graph=socfb-Reed98
  popd
fi

# Warm-up
pushd DeathStarBench/socialNetwork
../wrk2/wrk -D exp -t 5 -c 5 -d 10 -L -s ./wrk2/scripts/social-network/compose-post.lua http://$SVC_URL/wrk2-api/post/compose -R 10

# Measure CPU and Memory
sudo python3 $TESTBED/scripts/utils/cpumem_stats.py network_$1 > $TESTBED/logs/python.log 2>&1 &

# Run queries to log timings
../wrk2/wrk -D exp -t 10 -c 20 -d 60 -L -s ./wrk2/scripts/social-network/mixed-workload.lua http://$SVC_URL -R 500 >> $TESTBED/out/time_network_$1.run 2>&1

# Kill CPU and Memory measurement
pid=$(pgrep -f cpumem)
sudo kill -SIGINT $pid

popd