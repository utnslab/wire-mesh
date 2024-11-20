#!/bin/bash
# Run all the experiments for a particular service mesh.
# Arguments:
# 1: Name of the experiment
# 2: Scenario (istio/hypo/wire)
# 3: Directory to put the output files

# Check arguments
if [ $# -ne 3 ]; then
  echo "Usage: <script_name> <experiment_name> <scenario> <output_dir>"
  exit 1
fi

# Set the arguments
SCENARIO=$2
OUTPUT_DIR=$3

HOSTS=`./cloudlab/nodes.sh $1 0 4 --all`

# Get the control node (first line of $HOSTS)
CONTROL_NODE=$(echo $HOSTS | head -1 | awk '{print $1}')

# Construct the array of benchmark applications
APPS=("boutique" "reservation" "social")

# Iterate over the benchmark applications, and run experiments for each.
for APP in "${APPS[@]}"; do
  echo "Setting up application $APP for scenario $SCENARIO ..."
  ssh -o StrictHostKeyChecking=no $CONTROL_NODE "pushd \$HOME/scripts/deployment/$APP && ./deploy_p1.sh ${SCENARIO} && popd"

  echo "Running experiment for $APP ..."
  RATES=()
  if [[ "$SCENARIO" == "istio" ]]; then
    if [[ "$APP" == "boutique" ]]; then
      RATES=(50 100 120 150 180)
    elif [[ "$APP" == "reservation" ]]; then
      RATES=(100 500 1000 1500 1800 2000)
    elif [[ "$APP" == "social" ]]; then
      RATES=(100 500 1000 1500 1800 2000)
    fi
  elif [[ "$SCENARIO" == "hypo" ]]; then
    if [[ "$APP" == "boutique" ]]; then
      RATES=(50 100 150 200 225 250 280)
    elif [[ "$APP" == "reservation" ]]; then
      RATES=(100 500 1000 1500 2000 2400 2800 3000 3200)
    elif [[ "$APP" == "social" ]]; then
      RATES=(100 500 1000 1500 2000 2400 2800)
    fi
  elif [[ "$SCENARIO" == "wire" ]]; then
    if [[ "$APP" == "boutique" ]]; then
      RATES=(50 100 150 200 225 250 280)
    elif [[ "$APP" == "reservation" ]]; then
      RATES=(100 500 1000 1500 2000 2400 2800 3000 3200)
    elif [[ "$APP" == "social" ]]; then
      RATES=(100 500 1000 1500 2000 2400 2800 3000)
    fi
  fi

  # Run the experiments for the application for each rate
  for RATE in "${RATES[@]}"; do
    echo "Running experiment for $APP at rate $RATE ..."
    ./cloudlab/run_experiment.sh $1 $APP $SCENARIO $OUTPUT_DIR $RATE

    # Sleep for a minute between experiments
    sleep 60
  done

  # Tear down the application
  echo "Tearing down application $APP ..."
  if [[ "$APP" == "boutique" ]]; then
    ssh -o StrictHostKeyChecking=no $CONTROL_NODE "pushd \$HOME/scripts/deployment/$APP && kubectl delete -Rf yaml && popd"
  elif [[ "$APP" == "reservation" ]]; then
    ssh -o StrictHostKeyChecking=no $CONTROL_NODE "pushd \$HOME/DeathStarBench/hotelReservation && kubectl delete -Rf kubernetes && popd"
  elif [[ "$APP" == "social" ]]; then
    ssh -o StrictHostKeyChecking=no $CONTROL_NODE "helm uninstall social-network"
  fi

done