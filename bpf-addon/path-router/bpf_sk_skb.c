#include <stdio.h>

#include <linux/bpf.h>
#include <sys/socket.h>

#include <bpf/bpf_endian.h>
#include <bpf/bpf_helpers.h>

char LICENSE[] SEC("license") = "GPL";

// Define Sequences that are not relevant to the trace.
#define HTTP_HEX 0x48545450
#define GET_HEX 0x47455420
#define POST_HEX 0x504F5354
#define PRI_HEX 0x50524920
// Fixed size of the path. First element for path length, and remaining 10 for the path itself.
#define MAX_PATH_LEN 101

// path_map is a hash map that maps a trace_id to a path
// The key is a 64-bit integer that corresponds to the first 64 bits of the trace_id.
// The value is an array of 11 integers that corresponds to the path.
struct {
  __uint(type, BPF_MAP_TYPE_HASH);
  __uint(max_entries, 8192);
  __type(key, __u64);
  __type(value, __u8[MAX_PATH_LEN]);
  __uint(pinning, LIBBPF_PIN_BY_NAME);
} path_map SEC(".maps");

// eviction_map stores the trace ids in an array.
// This is used to evict the oldest trace id when the path_map is full.
struct {
  __uint(type, BPF_MAP_TYPE_ARRAY);
  __uint(max_entries, 8192);
  __type(key, int);
  __type(value, __u64);
} eviction_map SEC(".maps");

// circular_index_map stores the head and tail of the ring buffer in eviction_map.
// First element is the head and second element is the tail.
// Eviction happens at the head and insertion happens at the tail.
struct {
  __uint(type, BPF_MAP_TYPE_ARRAY);
  __uint(max_entries, 2);
  __type(key, int);
  __type(value, __u32);
} circular_index_map SEC(".maps");

// Long long to network byte order and vice versa.
#define HTONLL(x) ((1==bpf_htonl(1)) ? (x) : (((__u64)bpf_htonl((x) & 0xFFFFFFFFUL)) << 32) | bpf_htonl((__u32)((x) >> 32)))
#define NTOHLL(x) ((1==bpf_ntohl(1)) ? (x) : (((__u64)bpf_ntohl((x) & 0xFFFFFFFFUL)) << 32) | bpf_ntohl((__u32)((x) >> 32)))

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

SEC("sk_skb/stream_parser")
int parse_skb(struct __sk_buff *skb) {
  int err;
  unsigned long len = (long)skb->data_end - (long)skb->data;

  if (len < skb->len) {
    err = bpf_skb_pull_data(skb, skb->len);
    if (err < 0) {
      bpf_printk("ERR: bpf_skb_pull_data failed\n");
    }
  }

  return skb->len;
}

SEC("sk_skb/stream_verdict")
int read_skb(struct __sk_buff *skb) {
  // Bounds check to make verifier happy
  void *data = (void *)(long) skb->data;
  void *data_end = (void *)(long) skb->data_end;

  // Read the first 4 bytes of the message to check if the frame is of type 0x0A.
  __u32 first_4_bytes = 0;
  if (data + 4 > data_end) {
    return SK_PASS;
  }
  __builtin_memcpy(&first_4_bytes, data, 4);
  first_4_bytes = bpf_ntohl(first_4_bytes);

  if (first_4_bytes == HTTP_HEX || first_4_bytes == GET_HEX || first_4_bytes == POST_HEX || first_4_bytes == PRI_HEX) {
    // bpf_printk("Not a HTTP/2 request");
    return SK_PASS;
  }

  int frame_type = first_4_bytes & 0x000000FF;

  // Check if the frame is of type 0x0A -- this is the additional CTX frame.
  if (frame_type != 0x0A) {
    return SK_PASS;
  }

  // First 9B are frame headers - to be ignored.
  // The next 8B are the trace_id. And the next 44B are the path.
  void* bytes_start = data + 9;
  if (bytes_start + sizeof(__u64) > data_end) {
    return SK_PASS;
  }

  __u64 trace_id = 0;
  __builtin_memcpy(&trace_id, bytes_start, sizeof(__u64));
  
  trace_id = NTOHLL(trace_id);

  // Get the next (sizeof(__u8) * MAX_PATH_LEN) and save in the map.
  bytes_start = data + 17;
  if (bytes_start + MAX_PATH_LEN * sizeof(__u8) > data_end) {
    return SK_PASS;
  }

  __u8 path[MAX_PATH_LEN];
  __builtin_memcpy(&path, bytes_start, MAX_PATH_LEN * sizeof(__u8));

  // // Convert each element of the path to host byte order.
  // for (int i = 0; i < MAX_PATH_LEN; i++) {
  //   path[i] = bpf_ntohl(path[i]);
  // }

  // Save the path in the map.
  bpf_map_update_elem(&path_map, &trace_id, &path, BPF_ANY);

  // Get the head and tail of the ring buffer.
  int circular_index = 0;
  __u32* head_ptr = bpf_map_lookup_elem(&circular_index_map, &circular_index);
  circular_index = 1;
  __u32* tail_ptr = bpf_map_lookup_elem(&circular_index_map, &circular_index);

  // If the head and tail are not found in the map, return.
  if (!head_ptr || !tail_ptr) {
    return SK_PASS;
  }

  // Insert the trace_id in the eviction map.
  bpf_map_update_elem(&eviction_map, tail_ptr, &trace_id, BPF_ANY);
  *tail_ptr = (*tail_ptr + 1) % 8192;

  // If the eviction map is full, evict the oldest trace_id.
  if (*head_ptr == *tail_ptr) {
    // Remove the trace_id at the head of the ring buffer.
    __u64* trace_id_ptr = bpf_map_lookup_elem(&eviction_map, head_ptr);
    if (trace_id_ptr) {
      bpf_map_delete_elem(&path_map, trace_id_ptr);
    }

    // Increment the head of the ring buffer.
    *head_ptr = (*head_ptr + 1) % 8192;
  }

  bpf_printk("sk_skb -> Updated path information in the map");

  return SK_PASS;
}
