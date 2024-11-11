USER_TARGETS := load_sockops load_grpc_skmsg load_thrift_skmsg read_maps_skmsg read_maps_skb load_sk_skb
XDP_TARGETS := bpf_grpc_skmsg bpf_thrift_skmsg bpf_sockops bpf_sk_skb

COMMON_DIR = ../common

include $(COMMON_DIR)/common.mk

cleandir:
	rm -f $(USER_TARGETS)
	rm -f *.o
	rm -f *.ll