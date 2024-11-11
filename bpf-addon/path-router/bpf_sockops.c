#include <linux/in.h>
#include <linux/tcp.h>

#include <linux/bpf.h>
#include <sys/socket.h>

#include <bpf/bpf_endian.h>
#include <bpf/bpf_helpers.h>

char LICENSE[] SEC("license") = "GPL";

// sock_ops_map maps the sock_ops key to a socket descriptor
struct {
  __uint(type, BPF_MAP_TYPE_SOCKHASH);
  __uint(max_entries, 65535);
  __type(key, struct sock_key);
  __type(value, __u64);
} sock_ops_map SEC(".maps");

// `sock_key' is a key for the sockmap
struct sock_key {
  __u32 sip4;
  __u32 dip4;
  __u32 sport;
  __u32 dport;
} __attribute__((packed));

void print_ip4(unsigned int ip, char *msg) {
  unsigned char bytes[4];
  bytes[0] = ip & 0xFF;
  bytes[1] = (ip >> 8) & 0xFF;
  bytes[2] = (ip >> 16) & 0xFF;
  bytes[3] = (ip >> 24) & 0xFF;

  // Construct a string from the bytes
  const __u8 ipv4[] = {bytes[0], bytes[1], bytes[2], bytes[3]};

  // Print the string
  bpf_printk("%s %pi4", msg, ipv4);
}

// `sk_extract_key' extracts the key from the `bpf_sock_ops' struct
static inline void sk_extract_key(struct bpf_sock_ops *ops,
                                  struct sock_key *key) {
  key->dip4 = ops->remote_ip4;
  key->sip4 = ops->local_ip4;
  key->sport = (bpf_htonl(ops->local_port) >> 16);
  key->dport = ops->remote_port >> 16;
}

SEC("sockops")
int bpf_add_to_sockhash(struct bpf_sock_ops *skops) {
  __u32 family, op;

  family = skops->family;
  op = skops->op;

  switch (op) {
    case BPF_SOCK_OPS_PASSIVE_ESTABLISHED_CB:
    case BPF_SOCK_OPS_ACTIVE_ESTABLISHED_CB:
      if (family == AF_INET || family == AF_INET6) {
        struct sock_key key = {};
        sk_extract_key(skops, &key);

        // bpf_printk("Got new operation %d for socket of family %d.", op, family);
        int ret = bpf_sock_hash_update(skops, &sock_ops_map, &key, BPF_NOEXIST);
        if (ret != 0) {
          bpf_printk("skb -> Failed to update sockmap: %d", ret);
        // } else {
        //   bpf_printk("Added new socket to sockmap for source ip %u destination ip %u", key.sip4, key.dip4);
        //   print_ip4(key.sip4, "Added new socket to sockmap for source ip");
        //   print_ip4(key.dip4, "Added new socket to sockmap for destination ip");
        }
      }
      break;
    default:
      break;
  }
  return 0;
}
