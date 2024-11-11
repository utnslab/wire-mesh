# BPF Path Propagation
Repository for an eBPF-based add-on to parse messages, read the trace-id and add the Service IP to the corresponding outgoing message.
Includes implementations for HTTP/2 based gRPC communication and Thrift RPC communication.

## Working of Path Propagation

```
+-------------------------------------------------------------------+
|                         Service                                   |
+-------------------------------------------------------------------+
|         eBPF Program          |   |           eBPF Program        |
|           (sk_msg)            |   |             (sk_skb)          |
|  Read trace ID and path from  |   |  Fetch path from path_map for |
|  message and add to path_map  |   |  trace ID and add to message  |
+-------------------------------+   +-------------------------------+
```

The `sk_msg` program reads the trace ID from the incoming message and adds the path to the `path_map` for the trace ID.
Current implementation for gRPC looks for the `uber-trace-id` header in the request message (as this is one being propagated by [Jaeger Clients](https://www.jaegertracing.io/docs/1.63/client-libraries/) used in DeathStarBench). For other protocols, the corresponding trace ID header can be changed in the `bpf_grpc_skmsg` program.

## Usage

#### Compilation

Compile by executing the `Makefile` in `path-prop` directory:
```bash
$ cd path-prop
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
└── path-prop # Path propagation logic in eBPF, targetted for microservices using OpenTelemetry type library
```