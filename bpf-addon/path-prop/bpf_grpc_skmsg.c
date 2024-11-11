#include <stdio.h>
#include <string.h>

#include <linux/bpf.h>
#include <sys/socket.h>

#include <bpf/bpf_endian.h>
#include <bpf/bpf_helpers.h>

char LICENSE[] SEC("license") = "GPL";

// Maximum offset a header can be on.
#define MAX_OFFSET 2048
// Maximum number of headers to expect in the headers frame.
#define MAX_NUM_HEADERS 6
// Length of header name that we want to parse.
#define HEADER_NAME_LEN 9
// Length of header value that we want to store.
#define HEADER_VALUE_LEN 10
// Trace ID to match -- 8B is in network byte order.
#define TRACE_ID_8 0xb10a196cb2d832b6
#define TRACE_ID_1 0xa4
// Define Sequences that are not relevant to the trace.
#define HTTP_HEX 0x48545450
#define GET_HEX 0x47455420
#define POST_HEX 0x504F5354
#define PRI_HEX 0x50524920
// Fixed size of the path. First element for path length, and remaining 10 for the path itself.
#define MAX_PATH_LEN 101

// jmp_table is used to jump to the tail call function.
// Index 0 holds the tail program.
// Index 1 holds the extended parser.
struct {
  __uint(type, BPF_MAP_TYPE_PROG_ARRAY);
  __uint(max_entries, 2);
  __type(key, int);
  __type(value, __u32);
  __uint(pinning, LIBBPF_PIN_BY_NAME);
} jmp_table SEC(".maps");

// msg_arguments_map stores the arguments needed for tail calls.
// Index 0: holds the offset of the trace_id header in the message.
// Index 1: holds the frame_length of the message.
struct {
  __uint(type, BPF_MAP_TYPE_ARRAY);
  __uint(max_entries, 2);
  __type(key, int);
  __type(value, __u32);
} msg_arguments_map SEC(".maps");

// header_index_map stores the index of the trace_id header in the dynamic table.
// This is required to identify trace id headers in subsequent messages.
// Stored in a hash map with key as the destination IP address (stored as an integer).
struct {
  __uint(type, BPF_MAP_TYPE_HASH);
  __uint(max_entries, 65535);
  __type(key, __u32);
  __type(value, __u8);
} header_index_map SEC(".maps");

// path_map is a hash map that maps a trace_id to a path
// The key is a 64-bit integer that corresponds to the first 64 bits of the trace_id.
// The value is a 40B string that corresponds to the path.
// This map is pinned in the sk_skb program and must be read from there.
struct {
  __uint(type, BPF_MAP_TYPE_HASH);
  __uint(max_entries, 8192);
  __type(key, __u64);
  __type(value, __u8[MAX_PATH_LEN]);
  __uint(pinning, LIBBPF_PIN_BY_NAME);
} path_map SEC(".maps");

// svc_identifier_map stores the identifier of the current service.
// The map is updated by the loader program.
struct {
  __uint(type, BPF_MAP_TYPE_ARRAY);
  __uint(max_entries, 1);
  __type(key, int);
  __type(value, __u8);
  __uint(pinning, LIBBPF_PIN_BY_NAME);
} svc_identifier_map SEC(".maps");

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

SEC("sk_msg")
int propagate_path(struct sk_msg_md *msg) {
  bpf_printk("sk_msg -> Found TRACE_ID.");

  // Read offset of trace_id header from the map.
  int index = 0;
  __u32 *offset_ptr = bpf_map_lookup_elem(&msg_arguments_map, &index);
  if (!offset_ptr) {
    bpf_printk("ERR: offset_ptr not found");
    return SK_PASS;
  }

  // Bounds check to make verifier happy.
  __u32 offset = *offset_ptr;
  if (offset > MAX_OFFSET) {
    bpf_printk("ERR: offset out of bounds");
    return SK_PASS;
  }

  // Get the header index and save in state map.
  void* data = (void *)(long) msg->data;
  void* data_end = (void *)(long) msg->data_end;

  // Get the header index and save in state map.
  void* header_start = data + offset;
  
  // Read the first byte of the header to get the header type.
  if (header_start + 1 > data_end) {
    return SK_PASS;
  }
  __u8 header_index = *(unsigned char*)header_start;
  
  // The first two bits of the header index are to be ignored.
  header_index = header_index & 0x3F;
  // bpf_printk("Header index: %d", header_index);

  // Index is 0 if the header is not indexed yet. Non-zero otherwise.
  if (header_index == 0) {
    // Save the header index in state map.
    __u32 key = msg->remote_ip4;

    // All new entries are inserted at index 62.
    header_index = 62;
    bpf_map_update_elem(&header_index_map, &key, &header_index, BPF_ANY);

    // Since it is a new name, the header_start must be increase by (1 + name_length) bytes.
    // 1B for the name length and name_length bytes for the name.
    header_start += (1 + HEADER_NAME_LEN);
  }

  // Skip 1B for the header type and 1B for the value length.
  void* trace_id_start = header_start + 2;

  // Read the first 8B of the header value to get the trace_id.
  // This trace_id will be used to lookup into the path_map.
  __u64 trace_id = 0;
  if (trace_id_start + sizeof(__u64) > data_end) {
    return SK_PASS;
  }
  __builtin_memcpy(&trace_id, trace_id_start, sizeof(__u64));
  __u64 trace_id_hbo = NTOHLL(trace_id);

  // Read the path from the path_map.
  __u8 path[MAX_PATH_LEN] = {0};
  __u64 key = trace_id_hbo;
  __u8 *curr_path = bpf_map_lookup_elem(&path_map, &key);
  __u8 new_length = 1;
  if (curr_path) {
    __builtin_memcpy(path, curr_path, MAX_PATH_LEN * sizeof(__u8));
    for (int i = 1; i < MAX_PATH_LEN - 1; i++) {
      path[i+1] = path[i];
    }

    new_length = path[0] + 1;
  } else {
    bpf_printk("sk_msg -> Path not found in path_map.");
  }

  // Add the current service identifier to the path.
  int svc_identifier_index = 0;
  __u8 *svc_identifier_ptr = bpf_map_lookup_elem(&svc_identifier_map, &svc_identifier_index);
  if (!svc_identifier_ptr) {
    bpf_printk("ERR: svc_identifier_ptr not found");
    return SK_PASS;
  }
  __u8 svc_identifier = *svc_identifier_ptr;

  path[1] = svc_identifier;
  path[0] = new_length;

  // Insert a new frame, to add offset value.
  __u32 frame_len = sizeof(__u64) + MAX_PATH_LEN * sizeof(__u8) + 9;
  // bpf_printk("sk_msg -> Inserting new frame of length %d", frame_len);
  bpf_msg_push_data(msg, 0, frame_len, 0);

  // bpf_msg_push_data invalidates the pointers, so we need to get them again.
  data = (void *)(long) msg->data;
  data_end = (void *)(long) msg->data_end;

  // Write a new frame to the message.
  // Check if the message has enough space to write a new frame for the offset.
  if (data + frame_len > data_end) {
    bpf_printk("ERR: data + frame_len > data_end");
    return SK_PASS;
  }

  // First 4 bytes of the frame would be length of the frame (MAX_PATH_LEN * sizeof(__u8) - 9) B and type of the frame (0x0A).
  // Note that the 9 octets of the frame header are NOT included!
  // Shift frame_len by 8B to the left to make space for frame type.
  __u32 first_4_bytes = 0x0000000A;
  first_4_bytes = (frame_len - 9) << 8 | first_4_bytes;
  __u32 first_4_bytes_nbo = bpf_htonl(first_4_bytes);
  // bpf_printk("sk_msg -> first_4_bytes: %d", first_4_bytes);

  // Write the first 4 bytes of the frame.
  __builtin_memcpy(data, &first_4_bytes_nbo, 4);

  // Write all zeros for the next 5 bytes.
  __u8 zero = 0x00;
  __u32 zeros = 0x00000000;
  __builtin_memcpy(data + 4, &zeros, 4);
  __builtin_memcpy(data + 8, &zero, 1);

  // Write the 8B trace_id to the message - writing directly in network byte order.
  __builtin_memcpy(data + 9, &trace_id, sizeof(__u64));

  // Write the path to the message.
  for (int i = 0; i < MAX_PATH_LEN; i++) {
    // __u32 path_nbo = bpf_htonl(path[i]);
    __builtin_memcpy(data + 17 + i * sizeof(__u8), &path[i], sizeof(__u8));
  }

  // Remove the entry from the path map, corresponding to trace_id.
  bpf_map_delete_elem(&path_map, &key);

  bpf_printk("sk_msg -> Propagated path.");
  return SK_PASS;
}

SEC("sk_msg")
int parse_grpc_payload_extended(struct sk_msg_md *msg) {
  void *data = (void *)(long) msg->data;
  void *data_end = (void *)(long) msg->data_end;
  // bpf_printk("In parse_grpc_payload_extended");

  // Read offset of next header from the map.
  int index = 0;
  __u32 *offset_ptr = bpf_map_lookup_elem(&msg_arguments_map, &index);
  if (!offset_ptr) {
    bpf_printk("ERR: msg_offset_map not found");
    return SK_PASS;
  }

  // Bounds check to make verifier happy.
  __u32 offset = *offset_ptr;
  if (offset > MAX_OFFSET) {
    bpf_printk("ERR: offset out of bounds");
    return SK_PASS;
  }

  // Read frame_length of the message from the map.
  index = 1;
  __u32 *frame_length_ptr = bpf_map_lookup_elem(&msg_arguments_map, &index);
  if (!frame_length_ptr) {
    bpf_printk("ERR: msg_offset_map not found");
    return SK_PASS;
  }

  __u32 frame_length = *frame_length_ptr;
  if (frame_length > MAX_OFFSET) {
    bpf_printk("ERR: frame_length out of bounds");
    return SK_PASS;
  }

  void* header_start = data + offset;

  // Iteration variables for the loop below.
  __u8 name_length = 0;
  __u8 value_length = 0;
  __u8 header_type = 0;

  void* bytes = NULL;
  void* length_bytes = NULL;
  unsigned char length_char = 0;

  // Store read bytes.
  __u64 read_8_bytes = 0;
  __u8 read_1_byte = 0;

  // Check if an indexed header has already been registered.
  __u32 conn_key = msg->remote_ip4;
  __u8 *header_index_ptr = bpf_map_lookup_elem(&header_index_map, &conn_key);
  __u8 curr_header_index = 0;
  __u8 has_been_indexed = 0;

  // Dynamic table indexing starts at 62. And new entries are added at the start.
  // We need to check if the current header has already been indexed. Then, any
  // new entries imply that the current index should be incremented.
  if (!header_index_ptr) {
    curr_header_index = 62;
  } else {
    has_been_indexed = 1;
    curr_header_index = *header_index_ptr;
  }

  for (int i = 0; i < MAX_NUM_HEADERS; i++) {
    if (header_start > data + 9 + frame_length) {
      // We have reached the end of the header frame.
      // bpf_printk("Reached end of header frame");
      return SK_PASS;
    }

    // Reset the variables for the next header.
    name_length = 0;

    // Read the first byte of the header to get the header type.
    if (header_start + 1 > data_end) {
      bpf_printk("sk_msg cont -> ERR: header_start + 1 > data_end\n");
      return SK_PASS;
    }
    header_type = *(unsigned char*)header_start;

    if ((header_type & 0x80) == 0x80) {
      // Header is indexed header.
      // Skip 1B for header type and index.
      // bpf_printk("Found indexed header at %d", offset);
      header_start += 1;
      offset += 1;
    } else if ((header_type & 0xE0) == 0x20) {
      // Dynamic table size update.
      // Skip 1B for header type and table size.
      // bpf_printk("Found dynamic table size update at %d", offset);
      header_start += 1;
      offset += 1;
    } else {
      if (header_type == 0x10 || header_type == 0x00 || header_type == 0x40) {
        // Header is a new name.
        // bpf_printk("Found new name header at %d", offset);

        // It is a new name, so we need to read the length of the name.
        length_bytes = header_start + 1;
        if (length_bytes + 1 > data_end) {
          // bpf_printk("ERR: length_bytes + 1 > data_end %d", offset);
          return SK_PASS;
        }
        length_char = *(unsigned char*)length_bytes;
        name_length = length_char & 0x7F;

        // The header can be trace-id only if it is 9B long (Huffman coded "uber-trace-id").
        // Check if the header type starts with '01'.
        if (name_length == HEADER_NAME_LEN && (header_type & 0xC0) == 0x40) {
          // bpf_printk("Found literal header with incremental indexing");

          // Read the header name, and check if it is the trace id.
          // Header name is at 2B offset (1B for header type, 1B for name length)
          bytes = header_start + 2;
          if (bytes + HEADER_NAME_LEN > data_end) {
            // bpf_printk("ERR: bytes + HEADER_NAME_LEN > data_end %d", offset);
            return SK_PASS;
          }
          __builtin_memcpy(&read_8_bytes, bytes, sizeof(__u64));
          __builtin_memcpy(&read_1_byte, bytes + 8, sizeof(__u8));

          if (read_8_bytes == TRACE_ID_8 && read_1_byte == TRACE_ID_1) {
            // bpf_printk("Found trace id header at %d.", offset);
            // bpf_printk("Found trace id at %d", offset);

            // Put offset value of header start in the map.
            index = 0;
            bpf_map_update_elem(&msg_arguments_map, &index, &offset, BPF_ANY);

            // Make the tail call function.
            bpf_tail_call(msg, &jmp_table, 0);

            // Done with processing.
            return SK_PASS;
          }
        }

        // If the program hasn't returned yet, it means that the header is not the trace id.
        // Need to increment curr_header_index if the header has already been indexed.
        if (has_been_indexed == 1) {
          curr_header_index++;
          bpf_map_update_elem(&header_index_map, &conn_key, &curr_header_index, BPF_ANY);
        }

        // Increment 1B for name length and `name_length` bytes for name.
        // This increment is needed only for new name headers.
        header_start += (1 + name_length);
        offset += (1 + name_length);
      } else {
        // If header is indexed name, check whether an indexed literal header has been registered.
        if ((header_type & 0xC0) == 0x40) {
          header_index_ptr = bpf_map_lookup_elem(&header_index_map, &conn_key);

          if (header_index_ptr) {
            // Header has already been indexed -- check if the current header has the same index.
            __u8 header_index = header_type & 0x3F;
            if (header_index == *header_index_ptr) {
              // bpf_printk("Found indexed trace id header at %d.", offset);
              
              // Put offset of header start in the map.
              index = 0;
              bpf_map_update_elem(&msg_arguments_map, &index, &offset, BPF_ANY);

              // Make the tail call function.
              bpf_tail_call(msg, &jmp_table, 0);

              // Done with processing.
              return SK_PASS;
            }
          }
        }
      }

      // Skip 1B for header type and index.
      length_bytes = header_start + 1;
      if (length_bytes + 1 > data_end) {
        // bpf_printk("ERR: length_bytes + 1 > data_end %d", offset);
        return SK_PASS;
      }
      length_char = *(unsigned char*)length_bytes;
      value_length = length_char & 0x7F;

      // Skip a total of 1B for header type and index, 1B for value length, and `value_length` bytes for value.
      header_start += (2 + value_length);
      offset += (2 + value_length);
    }

    // bpf_printk("Completed header parsing after reading %d bytes", offset);
  }

  // bpf_printk("Completed parsing headers\n");
  return SK_PASS;
}

SEC("sk_msg")
int parse_grpc_payload(struct sk_msg_md *msg) {
  void *data = (void *)(long) msg->data;
  void *data_end = (void *)(long) msg->data_end;
  // unsigned long len = (long)msg->data_end - (long)msg->data;

  // Read the first 4B of the message into an integer and extract the length and type from it.
  __u32 first_4_bytes = 0;
  if (data + 4 > data_end) {
    // bpf_printk("ERR: data + 4 > data_end\n");
    return SK_PASS;
  }
  __builtin_memcpy(&first_4_bytes, data, 4);
  first_4_bytes = bpf_ntohl(first_4_bytes);

  // sk_msg captures HTTP/2 requests as well as HTTP/1.1 responses. Ignore HTTP/1.1 responses.
  // All HTTP/1.1 responses start with "HTTP" in ASCII, which is 0x48545450 in hex.
  // Also skip over GET, POST, and PRI headers. GET: 0x47455420 POST: 0x504f5354 PRI: 0x50524920
  if (first_4_bytes == HTTP_HEX || first_4_bytes == GET_HEX || first_4_bytes == POST_HEX || first_4_bytes == PRI_HEX) {
    // bpf_printk("Not a HTTP/2 request");
    return SK_PASS;
  }

  int frame_length = first_4_bytes >> 8;
  int frame_type = first_4_bytes & 0x000000FF;
  // bpf_printk("Processing message length: %d and frame length: %d", len, frame_length);

  // Only process further if the frame type is 0x01, which indicates the Header frame.
  if (frame_type != 0x01) {
    // bpf_printk("Frame type is not 0x01 %d\n", frame_type);
    return SK_PASS;
  }

  // Headers start after 9B of the frame.
  // (3B for length, 1B for type, 1B for flags, 4B for stream id)
  void* header_start = data + 9;
  int offset = 9;

  // Iteration variables for the loop below.
  int index = 0;
  __u8 name_length = 0;
  __u8 value_length = 0;
  __u8 header_type = 0;

  void* bytes = NULL;
  void* length_bytes = NULL;
  unsigned char length_char = 0;

  // Store read bytes.
  __u64 read_8_bytes = 0;
  __u8 read_1_byte = 0;

  // Check if an indexed header has already been registered.
  __u32 conn_key = msg->remote_ip4;
  __u8 *header_index_ptr = bpf_map_lookup_elem(&header_index_map, &conn_key);
  __u8 curr_header_index = 0;
  __u8 has_been_indexed = 0;

  // Dynamic table indexing starts at 62. And new entries are added at the start.
  // We need to check if the current header has already been indexed. Then, any
  // new entries imply that the current index should be incremented.
  if (!header_index_ptr) {
    curr_header_index = 62;
  } else {
    has_been_indexed = 1;
    curr_header_index = *header_index_ptr;
  }

  for (int i = 0; i < MAX_NUM_HEADERS; i++) {
    // Re-calculate frame length, in case it has changed.
    frame_length = first_4_bytes >> 8;

    if (header_start > data + 9 + frame_length) {
      // We have reached the end of the header frame.
      // bpf_printk("Reached end of header frame");
      return SK_PASS;
    }

    // Reset the variables for the next header.
    name_length = 0;

    // Read the first byte of the header to get the header type.
    if (header_start + 1 > data_end) {
      bpf_printk("sk_msg -> ERR: header_start + 1 > data_end\n");
      return SK_PASS;
    }
    header_type = *(unsigned char*)header_start;

    if ((header_type & 0x80) == 0x80) {
      // Header is indexed header.
      // Skip 1B for header type and index.
      // bpf_printk("Found indexed header at %d", offset);
      header_start += 1;
      offset += 1;
    } else if ((header_type & 0xE0) == 0x20) {
      // Dynamic table size update.
      // Skip 1B for header type and table size.
      // bpf_printk("Found dynamic table size update at %d", offset);
      header_start += 1;
      offset += 1;
    } else {
      if (header_type == 0x10 || header_type == 0x00 || header_type == 0x40) {
        // Header is a new name.
        // bpf_printk("Found new name header at %d", offset);

        // It is a new name, so we need to read the length of the name.
        length_bytes = header_start + 1;
        if (length_bytes + 1 > data_end) {
          // bpf_printk("ERR: length_bytes + 1 > data_end %d", offset);
          return SK_PASS;
        }
        length_char = *(unsigned char*)length_bytes;
        name_length = length_char & 0x7F;

        // The header can be trace-id only if it is 9B long (Huffman coded "uber-trace-id").
        // Check if the header type starts with '01'.
        if (name_length == HEADER_NAME_LEN && (header_type & 0xC0) == 0x40) {
          // bpf_printk("Found literal header with incremental indexing");

          // Read the header name, and check if it is the trace id.
          // Header name is at 2B offset (1B for header type, 1B for name length)
          bytes = header_start + 2;
          if (bytes + HEADER_NAME_LEN > data_end) {
            // bpf_printk("ERR: bytes + HEADER_NAME_LEN > data_end %d", offset);
            return SK_PASS;
          }
          __builtin_memcpy(&read_8_bytes, bytes, sizeof(__u64));
          __builtin_memcpy(&read_1_byte, bytes + 8, sizeof(__u8));

          if (read_8_bytes == TRACE_ID_8 && read_1_byte == TRACE_ID_1) {
            // bpf_printk("Found trace id header at %d.", offset);
            // bpf_printk("Found trace id at %d", offset);

            // Put offset value of header start in the map.
            bpf_map_update_elem(&msg_arguments_map, &index, &offset, BPF_ANY);

            // Make the tail call function.
            bpf_tail_call(msg, &jmp_table, 0);

            // Done with processing.
            return SK_PASS;
          }
        }

        // If the program hasn't returned yet, it means that the header is not the trace id.
        // Need to increment curr_header_index if the header has already been indexed.
        if (has_been_indexed == 1) {
          curr_header_index++;
          bpf_map_update_elem(&header_index_map, &conn_key, &curr_header_index, BPF_ANY);
        }

        // Increment 1B for name length and `name_length` bytes for name.
        // This increment is needed only for new name headers.
        header_start += (1 + name_length);
        offset += (1 + name_length);
      } else {
        // If header is indexed name, check whether an indexed literal header has been registered.
        if ((header_type & 0xC0) == 0x40) {
          header_index_ptr = bpf_map_lookup_elem(&header_index_map, &conn_key);

          if (header_index_ptr) {
            // Header has already been indexed -- check if the current header has the same index.
            __u8 header_index = header_type & 0x3F;
            if (header_index == *header_index_ptr) {
              // bpf_printk("Found indexed trace id header at %d.", offset);
              
              // Put offset of header start in the map.
              index = 0;
              bpf_map_update_elem(&msg_arguments_map, &index, &offset, BPF_ANY);

              // Make the tail call function.
              bpf_tail_call(msg, &jmp_table, 0);

              // Done with processing.
              return SK_PASS;
            }
          }
        }
      }

      // Skip 1B for header type and index.
      length_bytes = header_start + 1;
      if (length_bytes + 1 > data_end) {
        // bpf_printk("ERR: length_bytes + 1 > data_end %d", offset);
        return SK_PASS;
      }
      length_char = *(unsigned char*)length_bytes;
      value_length = length_char & 0x7F;

      // Skip a total of 1B for header type and index, 1B for value length, and `value_length` bytes for value.
      header_start += (2 + value_length);
      offset += (2 + value_length);
    }

    // bpf_printk("Completed header parsing after reading %d bytes", offset);
  }

  // For loop finished without returning -- call the extended function.
  // Add frame_length and offset in arguments map.
  index = 0;
  bpf_map_update_elem(&msg_arguments_map, &index, &offset, BPF_ANY);

  index = 1;
  bpf_map_update_elem(&msg_arguments_map, &index, &frame_length, BPF_ANY);
  bpf_tail_call(msg, &jmp_table, 1);

  return SK_PASS;
}
