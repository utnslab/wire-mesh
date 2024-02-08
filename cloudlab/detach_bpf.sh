#!/usr/bin/env bash
# 1: Name of the experiment
# 2: Start node of experiment
# 3: End node of experiment

NODE_PREFIX="node-"
EXP_NAME=$1
PROJECT_EXT="wisr-PG0"
DOMAIN="utah.cloudlab.us"
USER_NAME="dsaxena"
HOSTS=$(./cloudlab/nodes.sh $1 $2 $3)

# Run command on Control Node
CONTROL_NODE=$(echo $HOSTS | head -1 | awk '{print $1}')

ssh -o StrictHostKeyChecking=no ${CONTROL_NODE} "cd \$HOME/scripts/bpf; ./all_services.sh --control --detach"

# Run command on every node except the control node
for host in $HOSTS; do
  if [[ $host == $CONTROL_NODE ]]; then
    continue
  fi

  echo "Detaching all programs on $host"
  ssh -o StrictHostKeyChecking=no $host "cd \$HOME/scripts/bpf; ./all_services.sh --detach"
done
