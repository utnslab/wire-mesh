#include <stdio.h>
#include <string.h>

#include <linux/bpf.h>
#include <sys/socket.h>

#include <bpf/bpf_endian.h>
#include <bpf/bpf_helpers.h>

char LICENSE[] SEC("license") = "GPL";

// Upper bound for thrift header length
#define MAX_THRIFT_HEADER_LEN 256

// msg_id_counter stores the next message id to be assigned.
struct {
  __uint(type, BPF_MAP_TYPE_ARRAY);
  __uint(max_entries, 1);
  __type(key, int);
  __type(value, long);
} msg_id_counter SEC(".maps");

// msg_data_map carries a key-value pair of (msg_id, request_id), and can record
// upto 65535 messages at once.
struct {
  __uint(type, BPF_MAP_TYPE_HASH);
  __uint(max_entries, 65535);
  __type(key, int);
  __type(value, __u64);
} msg_data_map SEC(".maps");

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

SEC("sk_msg")
int parse_thrift_payload(struct sk_msg_md *msg) {
  void *data = (void *)(long) msg->data;
  void *data_end = (void *)(long) msg->data_end;
  unsigned long len = (long)msg->data_end - (long)msg->data;

  // Check if the message is valid.
  if (len < 0) {
    bpf_printk("ERR: Message length is negative\n");
    print_ip4(msg->local_ip4, "Local/Source IP:");
    print_ip4(msg->remote_ip4, "Remote/Dest IP:");
    return SK_PASS;
  }

  // Read the first 4B of the message to get the message type.
  unsigned char preface[sizeof(int)] = {0};
  if (data + sizeof(int) > data_end) {
    bpf_printk("ERR: data + sizeof(int) > data_end\n");
    return SK_PASS;
  }
  __builtin_memcpy(preface, data, sizeof(int));

  // If less than 0, then it is a Thrift message, else it is a Thrift frame.
  void* frame_start = data;
  int frame_length = bpf_ntohl(*(int*)preface);
  if (frame_length > 0) {
    // It is a Thrift frame: first 4B are length, and subsequent 4B are the preface.
    frame_start = data + 4;
  }

  // Check if the 4th byte is 0x01, which indicates a Thrift CALL message.
  if (frame_start + 4 > data_end) {
    bpf_printk("ERR: frame_start + 4 > data_end\n");
    return SK_PASS;
  }

  if (((unsigned char*)frame_start)[3] != 0x01) {
    return SK_PASS;
  }

  // Parse `data` to get the request id
  // First, get the length of Thrift header, located at offset 4
  // (2B for protocol-version, 1B unused, 1B for message-type)
  unsigned char header_length_bytes[sizeof(__u32)] = {0};

  // Bounds check to make verifier happy
  if (frame_start + 4 + sizeof(__u32) > data_end) {
    bpf_printk("ERR: frame_start + 4 + sizeof(__u32) > data_end\n");
    return SK_PASS;
  }
  __builtin_memcpy(header_length_bytes, frame_start + 4, sizeof(__u32));
  __u32 thrift_header_len = bpf_ntohl(*(__u32*)header_length_bytes);

  // Bounds check to make verifier happy
  if (thrift_header_len > MAX_THRIFT_HEADER_LEN) {
    bpf_printk("ERR: thrift_header_len > len %u\n", thrift_header_len);
    return SK_PASS;
  }

  // Now, get the frame_start field, located at offset 4 + 4 + thrift_header_len + 4
  // (4B for pre-header, 4B for length, thrift_header_len for header, 4B for sequence id)
  void* type_offset = frame_start + thrift_header_len + 12;
  if (type_offset + 1 > data_end) {
    bpf_printk("ERR: type_offset + 1 > data_end\n");
    return SK_PASS;
  }

  // Type 0x0a indicates a T_I64 integer type.
  if ((*(unsigned char*)type_offset) != 0x0a) {
    bpf_printk("ERR: type is not 0x0a\n");
    return SK_PASS;
  }

  // Now, get the request id, located at offset 4 + 4 + thrift_header_len + 4 + 1 + 2
  // (thrift_header_len + 12 for pre-frame_start, 1B for field type, 2B for field id)
  unsigned char request_id_bytes[sizeof(__u64)] = {0};
  
  // Bounds check to make verifier happy
  void* request_id_offset = frame_start + thrift_header_len + 15;
  if (request_id_offset + sizeof(__u64) > data_end) {
    bpf_printk("ERR: request_id_offset + sizeof(__u64) > data_end\n");
    return SK_PASS;
  }
  __builtin_memcpy(request_id_bytes, request_id_offset, sizeof(__u64));
  __u64 request_id = bpf_be64_to_cpu(*(__u64*)request_id_bytes);

  // Get the next message id
  int index = 0;
  int *msg_id_ptr;
  msg_id_ptr = bpf_map_lookup_elem(&msg_id_counter, &index);
  if (!msg_id_ptr) {
    bpf_printk("ERR: msg_id_counter not found\n");
  } else {
    // Store the data of the message in the map
    bpf_map_update_elem(&msg_data_map, msg_id_ptr, &request_id, BPF_ANY);

    // Increment the message id counter
    *msg_id_ptr += 1;
  }

  return SK_PASS;
}
