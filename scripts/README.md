## Scripts Usage

### Directory Structure

```
.
├── deployment/
├── install_docker.sh
├── mesh_setup.sh
├── mesh_uninstall.sh
├── plots/
├── README.md
└── xdp_setup.sh
```

`deployment/` directory contains all deployment scripts for various applications.
Currently, it has the `bookinfo` and `boutique` microservice applications.

`plots/` directory contains plotting scripts.

### Usage

**All scripts must be run from the `scripts` directory!!**

1. Setup:  
  Run `mesh_setup.sh <I/L/P>` (I: Istio, L: Linkerd, P: No Mesh) to install the respective mesh and configure the cluster to use the mesh.
2. Running Queries:  
  For each application, the `run_query_<mesh>.sh <init>` command will run index page queries for the cluster. Here:  
  `<mesh>`: `istio`, `linkerd`, or `plain`  
  `<init>`: can be 1 if this is the first time running for this application, in which case it will spawn the microservice graph, set up ingress controller and expose a URL to access the application. 0 if the microservice is already set up and the query is being re-run.