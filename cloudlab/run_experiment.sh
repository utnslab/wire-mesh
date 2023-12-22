#!/usr/bin/env bash
# Arguments:
# 1: Name of the experiment
# 2: Name of application
# 3: Service mesh name
# 4: Directory to put the output files
# 5: Request rate to run the experiment at

# Check arguments
if [ $# -ne 5 ]; then
  echo "Usage: <script_name> <experiment_name> <application_name> <service_mesh_name> <output_dir> <rate>"
  exit 1
fi

START_NODE=0
END_NODE=3

RAW_HOSTS=$(./cloudlab/nodes.sh $1 0 3)
CLIENT_HOST=$(./cloudlab/nodes.sh $1 4 4)
readarray -t HOSTS   <<<"$RAW_HOSTS"

APPL=$2
MESH=$3
RATE=$5

# List of ip addresses of the nodes
IP_ADDR=(10.10.1.1 10.10.1.2 10.10.1.3 10.10.1.4)

# CI code to start the experiment
./cloudlab/ci.sh $1 0 4 0

# Need to set a flag if MESH is istio and APPL is boutique or bookinfo
INGRESS=0
if [[ "$MESH" == "istio" ]] && [[ "$APPL" == "boutique" || "$APPL" == "bookinfo" ]]; then
  INGRESS=1
fi

# If INGRESS is set, then get the INGRESS_PORT from the control node.
if [[ $INGRESS -eq 1 ]]; then
  INGRESS_PORT=$(ssh -o StrictHostKeyChecking=no ${HOSTS[0]} "kubectl get svc istio-ingressgateway -n istio-system -o jsonpath='{.spec.ports[?(@.name==\"http2\")].nodePort}'")
  echo "Ingress external port is $INGRESS_PORT"
fi

# Start run_query.sh on the control node
echo "Starting run_query on ${HOSTS[0]} ${IP_ADDR[0]} ..."
ssh -o StrictHostKeyChecking=no ${HOSTS[0]} "tmux new-session -d -s run_query \"
  pushd \$HOME/scripts/deployment/$APPL &&
  ./run_query.sh --mesh $MESH -s -I ${IP_ADDR[0]} &&
  popd\""

# Start stats server on all the other nodes
for ((i = 1; i < ${#HOSTS[@]}; i++)); do
  echo "Starting stats server on ${HOSTS[$i]} ${IP_ADDR[$i]} ..."
  ssh -o StrictHostKeyChecking=no ${HOSTS[$i]} "tmux new-session -d -s stats_server \"
    sudo python3 \$HOME/scripts/utils/stats_server.py ${IP_ADDR[$i]}\""
done

# Start run_query client on the client node
echo "Starting run_query client on ${CLIENT_HOST} ..."

# If INGRESS is set, then use the INGRESS_PORT
if [[ $INGRESS -eq 1 ]]; then
  ssh -o StrictHostKeyChecking=no ${CLIENT_HOST} "tmux new-session -d -s run_query \"
    pushd \$HOME/scripts/deployment/$APPL &&
    rm -f \$HOME/out/time_${APPL}_${MESH}.run &&
    ./run_query.sh --mesh $MESH -c -I ${IP_ADDR[0]} --port ${INGRESS_PORT} -r ${RATE} &&
    popd\""
else
  ssh -o StrictHostKeyChecking=no ${CLIENT_HOST} "tmux new-session -d -s run_query \"
    pushd \$HOME/scripts/deployment/$APPL &&
    rm -f \$HOME/out/time_${APPL}_${MESH}.run &&
    ./run_query.sh --mesh $MESH -c -I ${IP_ADDR[0]} -r ${RATE} &&
    popd\""
fi

# Wait for the stats to be completed.
sleep 2m

# Wait for user to press enter
read -p "Press enter to stop the experiment and get stats ..."

# Get the stats from each of the nodes
mkdir -p $4/${APPL}-${MESH}-${RATE}-$(date +%d.%m-%H:%M)
cp scripts/deployment/$APPL/run_query.sh $4/${APPL}-${MESH}-${RATE}-$(date +%d.%m-%H:%M)/
for ((i = 0; i < ${#HOSTS[@]}; i++)); do
  echo "Getting stats from ${HOSTS[$i]} ..."
  scp -o StrictHostKeyChecking=no ${HOSTS[$i]}:~/out/stats_${APPL}_${MESH}.pkl $4/${APPL}-${MESH}-${RATE}-$(date +%d.%m-%H:%M)/stats_${APPL}_${MESH}_$i.pkl
done

# Get time from the client node
echo "Getting time from ${CLIENT_HOST} ..."
scp -o StrictHostKeyChecking=no ${CLIENT_HOST}:~/out/time_${APPL}_${RATE}_${MESH}.run $4/${APPL}-${MESH}-${RATE}-$(date +%d.%m-%H:%M)/