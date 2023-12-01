#!/usr/bin/env bash
# Arguments:
# 1: Name of the experiment
# 2: Start node
# 3: End node

HOSTS=`./cloudlab/nodes.sh $1 $2 $3 --all`

# Check if the nodes are reachable via a SSH command every 1 minute
while [ 1 ]; do
  FLAG=0
  for host in $HOSTS; do
    if ! (ssh -o StrictHostKeyChecking=no $host "echo 'test' > /dev/null" 2>/dev/null) ; then
      echo "Waiting for $host to come up ..."
      FLAG=1
      sleep 1m
      break
    else
      echo "Node $host is up"
    fi
  done
  if [ $FLAG -eq 0 ]; then
    break
  fi
done