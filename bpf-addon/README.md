# BPF Path Propagation
Repository for an eBPF-based add-on to parse messages, read the trace-id and add the Service IP to the corresponding outgoing message.
Includes implementations for HTTP/2 based gRPC communication and Thrift RPC communication.

## Usage

#### Compilation

Compile by executing the `Makefile` in `path-router` directory:
```bash
$ cd path-router
$ make
```

#### Installation

To install the eBPF add-on, the pods need to be started. And the first connection should be established *after* the eBPF programs have been installed.
Install the eBPF add-ons at all pods using (`$WIRE_ROOT` is the root directory of the wire repository):

```bash
$ cd $WIRE_ROOT
$ ./scripts/bpf/attach_bpf_service.sh --service <service-name> --ops --skb --skmsg --control
```

## Directory Structure

```bash
.
├── README.md
├── common      # Common headers and parsing files (clone of https://github.com/xdp-project/xdp-tutorial/tree/master/common)
├── headers     # Necessary bpf headers included in bpf programs
├── lib         # libbpf library that contains bpf helper functions used by the eBPF programs
└── path-router # Path propagation logic in eBPF, targetted for microservices using OpenTelemetry type library
```