#!/usr/bin/env bash
# Run query for the Social Network benchmark with different service meshes
# Arguments:
# --mesh: service mesh name
# --init: whether first time install
# --server: whether to start the stats server
# --client: whether to start the stats client
# --ip: IP address of the stats server

showHelp() {
cat << EOF  
Usage: <script_name> -m <mesh-name> [-isc] [-I <ip>] [-r <rate>]

Run query for the Social Network benchmark with different service meshes

-h, -help,      --help        Display help
-m, -mesh,      --mesh        Service mesh name to put in the output file
-i, -init,      --init        Whether first time install
-s, -server,    --server      Whether to start the stats server
-c, -client,    --client      Whether to start the stats client
-I, -ip,        --ip          IP address of the stats server
-r, -rate,      --rate        Rate of requests per second

EOF
}

MESH=""
INIT=0
SERVER=0
CLIENT=0
IP=""
RATE=2000

options=$(getopt -l "help,mesh:,init,server,client,ip:" -o "hm:iscI:r:" -a -- "$@")

eval set -- "$options"

while true; do
  case "$1" in
  -h|--help) 
      showHelp
      exit 0
      ;;
  -m|--mesh)
      shift
      MESH=$1
      ;;
  -i|--init)
      INIT=1
      ;;
  -c|--client)
      CLIENT=1
      ;;
  -s|--server)
      SERVER=1
      ;;
  -I|--ip)
      shift
      IP=$1
      ;;
  -r|--rate)
      shift
      RATE=$1
      ;;
  --)
      shift
      break;;
  esac
  shift
done

: "${TESTBED:=$HOME}"
pushd $TESTBED

if [[ $INIT -eq 1 ]]; then
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

if [[ $SERVER -eq 1 ]]; then
  # Start the stats server
  echo "Starting the stats server with IP=$IP"
  sudo python3 $TESTBED/scripts/utils/stats_server.py $IP > $TESTBED/logs/python.log 2>&1 &
  wait
fi

if [[ $CLIENT -eq 1 ]]; then
  GATEWAY_URL="$IP:32000"
  if [[ $SERVER -eq 1 ]]; then
    echo "Setting it to kubectl service endpoint"
    INGRESS_HOST=$(kubectl get svc nginx-thrift -o jsonpath='{.spec.clusterIP}')
    INGRESS_PORT=$(kubectl get svc nginx-thrift -o jsonpath='{.spec.ports[?(@.protocol=="TCP")].port}')

    GATEWAY_URL=$INGRESS_HOST:$INGRESS_PORT
  fi

  echo "Starting the test with GATEWAY_URL=$GATEWAY_URL"

  # Warm-up
  pushd DeathStarBench/socialNetwork
  ../wrk2/wrk -D exp -t 5 -c 5 -d 10 -L -s ./wrk2/scripts/social-network/compose-post.lua http://$GATEWAY_URL/wrk2-api/post/compose -R 10

  # Start the stats client
  sudo python3 $TESTBED/scripts/utils/stats_client.py $TESTBED/scripts/utils/config social_$MESH > $TESTBED/logs/python.log 2>&1 &

  sleep 30

  # Run queries to log timings
  ../wrk2/wrk -D exp -t 10 -c 10 -d 60 -L -s ./wrk2/scripts/social-network/mixed-workload.lua http://$GATEWAY_URL -R $RATE >> $TESTBED/out/time_social_${RATE}_${MESH}.run 2>&1

  # Kill CPU and Memory measurement
  pid=$(pgrep -f stats_client)
  sudo kill -SIGINT $pid
fi

popd