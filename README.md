# Wire Mesh

The repository for the Wire Mesh project.

## Setup Instructions

Running the Wire Mesh only requires a Linux version higher than 5.15, a running Kubernetes cluster and a few Python packages that can be installed using:
```bash
cd scripts
./setup.sh
```

For a Cloudlab cluster, we provide a simple script to set up the environment.
The setup involves running the scripts from the local machine and then testing the setup on the Cloudlab cluster.

### Setup for Cloudlab

1. On your local machine, set the Cloudlab environment variables for CLI:
   ```bash
    export CLOUDLAB_USERNAME=<cloudlab-username>
    export CLOUDLAB_PROJECT=<cloudlab-project>
    export CLOUDLAB_CLUSTER=<cloudlab-cluster-domain>
   ```
   `CLOUDLAB_USERNAME` is the username under which you are SSH-ing into the Cloudlab cluster, `CLOUDLAB_PROJECT` is the project under which the Cloudlab cluster is registered (both of these can be found on the Cloudlab experiment page).
   The domain for the cluster is usually like `utah.cloudlab.us` or `wisc.cloudlab.us`, etc.

2. On the local machine, start the setup script. This will SSH into the cloudlab nodes using the above environment variables and set up the Kubernetes cluster:
   ```bash
    ./cloudlab/config.sh <cloudlab-experiment-name> 0 3 1 && ./cloudlab/client_config.sh <cloudlab-experiment-name> 4
   ```

3. On Cloudlab node 0, check that the Kubernetes cluster is up and running:
   ```bash
    kubectl get nodes
   ```
   The output should show the nodes in the cluster in the `Ready` state.

4. Again from the local machine, push the necessary code on to the Cloudlab cluster:
   ```bash
    ./cloudlab/ci.sh <cloudlab-experiment-name> 0 3 1
   ```

## Running the Wire Mesh

On the Kubernetes control node (for Cloudlab setup using [the above](#setup-for-cloudlab) steps, this should be node0), run:

1. Install a service mesh:
   ```bash
   cd scripts/
   ./mesh_setup.sh --mesh <mesh>
   ```
   where `<mesh>` is the service mesh to install (e.g., `wire`, `istio`, `cilium`). 

2. Start an application on the Kubernetes cluster (this repository contains scripts to set up four applications in `scripts/deployment`):
   ```bash
   cd scripts/deployment/reservation
   ./run_query.sh --init --mesh <mesh>
   ```
   where `<mesh>` is the service mesh to use for this (e.g., `wire`, `istio`, `cilium`).
   This should start the application -- check by running `kubectl get pods` after a few seconds. It should show the pods in the `Running` state.

3. Run the workload generator:
    ```bash
    cd scripts/depoloyment/reservation
    ./run_query.sh --mesh <mesh> -c -I <ip-addr> -r <rate>
    ```
    where `<ip-addr>` is the IP address of the control node, and `<rate>` is the request rate to generate.

The above should save results in the `$HOME/out` directory.

## Reproducing results in the paper

### Latency-Throughput and CPU-Memory usage results
Setup the cluster using instruction [above](#setup-instructions).
Then, follow the instructions below to get the necessary traces and logs for Figures 9 and 10 in the paper.

#### End-to-end experiments

Running the end-to-end experiments for a policy, require execution of the policy for the given scenario over several traces, for all benchmark applications.
This can take a long time (up to two hours) to generate all results.
In order to only test the policy on a single trace, see the instructions in [this section](#single-run).

To generate all the necessary results, please repeat the following steps for `mesh`=`istio`,`hypo`, and `wire` and then, to plot the final figures, use the plotting scripts described in [this section](#plotting-scripts).

1. Set up a particular service mesh (on the control node, node0 of the Cloudlab cluster):
    ```bash
    cd scripts
    ./mesh_setup.sh --mesh <mesh>
    ```
    where `<mesh>` is one of `istio`, `hypo` (shown in paper as `Istio++`) or `wire`.

2. Start the evaluation (from your local machine):
   ```bash
   ./cloudlab/run_eval_p1.sh <cloudlab-experiment-name> <scenario> <results-dir>
   ```
   where the scenario is one of `istio`, `hypo` (shown in paper as `Istio++`) or `wire`.

   To run the evaluation for policy P1+P2, run the following script:

   ```bash
   ./cloudlab/run_eval_p1p2.sh <cloudlab-experiment-name> <mesh> <results-dir>
   ```
   where the mesh is one of `istio`, `hypo` (shown in paper as `Istio++`) or `wire`.

3. Remove the mesh and the running containers (on the control node, node0 of the Cloudlab cluster):
   ```bash
   cd scripts
   ./mesh_uninstall.sh --mesh <mesh>
   ```

#### Single run

One can simply run a particular mesh (Istio/Istio++/Wire) for a single trace using the following script (note that for the complete results, one needs to repeat this for all benchmark applications and multiple request rates - this is done in the [end-to-end experiments](#end-to-end-experiments) above.):

1. Setup the application (on control node, node0 of the Cloudlab cluster):
   ```bash
   cd scripts/deployment/<app (reservation/social/boutique)>
   ./deploy_p1.sh <mesh>
   ```
   where `<mesh>` can be `istio`, `hypo` (shown in paper as `Istio++`) or `wire`.

2. Run the experiment (on local machine):
   ```bash
   ./cloudlab/run_experiment.sh <cloudlab-experiment-name> <app> <mesh> <results-dir> <request-rate>
   ```
   where `<app>` is one of `reservation`, `social`, `boutique`, `<mesh>` is one of `istio`, `hypo` (shown in paper as `Istio++`) or `wire`, and `<request-rate>` is the request rate to generate.
   The script will run a workload with `<request-rate>` requests per second and save the observed latencies and CPU/Memory usage in `<results-dir>`.

#### Plotting Scripts

The results can be plotted by running the following scripts:

```bash
cd scripts/plots
python plot_tput_latency_apps.py <results-dir> boutique reservation social
python plot_cpumem_apps.py <results-dir> boutique reservation social
```

The final plotted figures will be saved in the `scripts/plots/figures` directory.

### Evaluation of control plane on production traces

The Wire control plane can be executed to find the optimal placement for production traces as follows:
    
```bash
cd pkg/placement
go test -timeout 9999s -v placement_test.go placement.go generate.go -run Production -args -logtostderr -traces ../../scripts/deployment/production/appls.json
```

The `appls.json` file contains details of the production traces from [Alibaba Cluster Data](https://github.com/alibaba/clusterdata).
This file was generated by running the `read_traces.py` followed by `analyze_traces.py` scripts in `scripts/deployment/production/` on the raw trace data.
