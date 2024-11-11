static const char *__doc__ = "User space program to load bpf program for the fast path\n";

#include <bpf/bpf.h>
#include <bpf/libbpf.h>
#include <errno.h>
#include <fcntl.h>
#include <string.h>
#include <unistd.h>

#include "../common/common_params.h"
#include "../common/common_user_bpf_xdp.h"

static const struct option_wrapper long_options[] = {

    {{"help", no_argument, NULL, 'h'}, "Show help", false},

    {{"map", no_argument, NULL, 'm'},
     "Which map should the fast path attach to (default: sock_ops_map)"},

    {{"cgroup", required_argument, NULL, 'c'},
     "Cgroup file to attach to (default: '' => /sys/fs/cgroup/unified)"},

    {{"service_ip", required_argument, NULL, 's'},
     "Service identifier IP"},

    {{"unload", no_argument, NULL, 'U'},
     "Unload sockops program instead of loading"},

    {{0, 0, NULL, 0}, NULL, false}};

const char *pin_dir = "/sys/fs/bpf";

int string_to_ip(char* ip_str, __u32* ip) {
  int a, b, c, d;
  int ret = sscanf(ip_str, "%d.%d.%d.%d", &a, &b, &c, &d);
  if (ret != 4) {
    fprintf(stderr, "ERR: parsing IP string\n");
    return -1;
  }
  *ip = a + (b << 8) + (c << 16) + (d << 24);
  return 0;
}

int main(int argc, char **argv) {
  int err, len;
  int subdir_len;
  int prog_fd, tail_fd, extended_fd;
  int sock_map_fd, jmp_table_fd, svcid_map_fd;
  int index = 0;

  __u8 svc_ip;
  char* dot;
  char map_filename[PATH_MAX];
  char jmp_table_filename[PATH_MAX];
  char svcid_map_filename[PATH_MAX];
  char prog_filename[PATH_MAX];
  char map_dir[PATH_MAX];
  char pod_identifier[PATH_MAX];

  struct bpf_object *fastpath_obj;
  struct bpf_program *fastpath_prog;
  struct bpf_program *propagate_prog;
  struct bpf_program *extended_prog;

  size_t fastpath_prog_len;
  const char *fastpath_file = "bpf_grpc_skmsg.o";

  struct config cfg = {
    .cgroup_name = "",
    .service_ip = "",
    .map_name = "sock_ops_map",
    .do_unload = false,
  };
  parse_cmdline_args(argc, argv, long_options, &cfg, __doc__);

  // <=============== Construct all the paths needed. ===============>
  // Remove '.' from cgroup name
  dot = strchr(cfg.cgroup_name, '.');
  subdir_len = dot ? dot - cfg.cgroup_name + 1 : PATH_MAX;

  // Use only the PodUID characters of cgroup name. 
  len = snprintf(pod_identifier, subdir_len, "%s", cfg.cgroup_name);
  if (len < 0) {
    fprintf(stderr, "ERR: constructing full mapname path\n");
    goto fail;
  }

  // Get fd of the pinned sockops map in pindir/subdir
  len = snprintf(map_filename, PATH_MAX, "%s/%s-%s/%s", pin_dir, pod_identifier, "sockops", cfg.map_name);
  if (len < 0) {
    fprintf(stderr, "ERR: constructing full mapname path\n");
    goto fail;
  }

  // Construct the map dir name for pinned maps.
  len = snprintf(map_dir, PATH_MAX, "%s/%s", pin_dir, pod_identifier);
  if (len < 0) {
    fprintf(stderr, "ERR: creating map dirname\n");
    goto fail;
  }

  // Construct the file name to get the pinned jmp table.
  len = snprintf(jmp_table_filename, PATH_MAX, "%s/%s", map_dir, "jmp_table");
  if (len < 0) {
    fprintf(stderr, "ERR: creating jmp table filename\n");
    goto fail;
  }

  // Construct the file name to get the pinned svc id map.
  len = snprintf(svcid_map_filename, PATH_MAX, "%s/%s", map_dir, "svc_identifier_map");
  if (len < 0) {
    fprintf(stderr, "ERR: creating svc id map filename\n");
    goto fail;
  }

  len = snprintf(prog_filename, PATH_MAX, "%s/%s-prog/%s", pin_dir, pod_identifier, "grpc_skmsg");
  if (len < 0) {
    fprintf(stderr, "ERR: constructing pin prog path\n");
    goto fail;
  }
  // <=========== All paths needed are constructed here. ===========>

  sock_map_fd = bpf_obj_get(map_filename);
  if (sock_map_fd < 0) {
    fprintf(stderr, "ERR: bpf_obj_get(%s): %s\n", map_filename, strerror(errno));
    goto fail;
  }

  // Check if the program is to be unloaded.
  if (cfg.do_unload) {
    // Get the fastpath program fd
    prog_fd = bpf_obj_get(prog_filename);
    if (prog_fd < 0) {
      fprintf(stderr, "ERR: bpf_obj_get(%s): %s\n", prog_filename, strerror(errno));
      goto fail;
    }

    // Detach the fastpath program from the map.
    err = bpf_prog_detach2(prog_fd, sock_map_fd, BPF_SK_MSG_VERDICT);
    if (err) {
      fprintf(stderr, "ERR: bpf_prog_detach2() failed: %s\n", strerror(errno));
      goto fail;
    }

    printf("Successfully unloaded fastpath program via detach2. Please manually remove any loaded maps.\n");
    return 0;
  }

  // Open, load and attach fastpath_obj if not already attached
	DECLARE_LIBBPF_OPTS(bpf_object_open_opts, opts,
			    .pin_root_path = map_dir);
  fastpath_obj = bpf_object__open_file(fastpath_file, &opts);
  if (libbpf_get_error(fastpath_obj)) {
    fprintf(stderr, "ERR: opening BPF-OBJ file (%s) (%s)\n", fastpath_file,
            strerror(-libbpf_get_error(fastpath_obj)));
    goto fail;
  }

  fastpath_prog = bpf_object__find_program_by_name(fastpath_obj, "parse_grpc_payload");
  if (!fastpath_prog) {
    fprintf(stderr, "ERR: finding program by name\n");
    goto fail;
  }

  fastpath_prog_len = bpf_program__insn_cnt(fastpath_prog);
  fprintf(stdout, "Number of instructions in program: %zu\n", fastpath_prog_len);

  // Load the sockops program
  err = bpf_object__load(fastpath_obj);
  if (err) {
    fprintf(stderr, "ERR: loading BPF-OBJ file %d (%s) (%s)\n", err, fastpath_file,
            strerror(-err));
    goto fail;
  }

  // Pin the fastpath program in the pin directory.
  err = bpf_program__pin(fastpath_prog, prog_filename);
  if (err) {
    fprintf(stderr, "ERR: bpf_program__pin(%s) failed: %s\n", prog_filename, strerror(errno));
    goto fail;
  }

  // Attach the sk_msg program to the map.
  // Using core BPF API as libbpf doesn't support sk_msg yet.
  err = bpf_prog_attach(bpf_program__fd(fastpath_prog), sock_map_fd, BPF_SK_MSG_VERDICT, 0);
  if (err) {
    fprintf(stderr, "ERR: bpf_program__attach() failed: %s\n", strerror(errno));
    goto fail;
  }

  // Load propagate tail called function.
  propagate_prog = bpf_object__find_program_by_name(fastpath_obj, "propagate_path");
  if (!propagate_prog) {
    fprintf(stderr, "ERR: finding program by name\n");
    goto fail;
  }

  // Get the fd of the propagate tail called function.
  tail_fd = bpf_program__fd(propagate_prog);

  // Load propagate tail called function.
  extended_prog = bpf_object__find_program_by_name(fastpath_obj, "parse_grpc_payload_extended");
  if (!propagate_prog) {
    fprintf(stderr, "ERR: finding program by name\n");
    goto fail;
  }

  // Get the fd of the propagate tail called function.
  extended_fd = bpf_program__fd(extended_prog);

  // Read the jmp table from the pinned file.
  jmp_table_fd = bpf_obj_get(jmp_table_filename);

  // Add the tail called function to the jmp table.
  index = 0;
  err = bpf_map_update_elem(jmp_table_fd, &index, &tail_fd, BPF_ANY);
  if (err) {
    fprintf(stderr, "ERR: bpf_map_update_elem() failed: %s\n", strerror(errno));
    goto fail;
  }

  // Add the extended function to the jmp table.
  index = 1;
  err = bpf_map_update_elem(jmp_table_fd, &index, &extended_fd, BPF_ANY);
  if (err) {
    fprintf(stderr, "ERR: bpf_map_update_elem() failed: %s\n", strerror(errno));
    goto fail;
  }

  // // Write the svc identifier to the svc_identifier_map.
  // err = string_to_ip(cfg.service_ip, &svc_ip);
  // if (err) {
  //   fprintf(stderr, "ERR: string_to_ip() failed");
  //   goto fail;
  // }

  // Get the svc_identifier_map fd.
  svcid_map_fd = bpf_obj_get(svcid_map_filename);

  // Add the svc identifier to the svc_identifier_map.
  index = 0;
  svc_ip = 0;
  err = bpf_map_update_elem(svcid_map_fd, &index, &svc_ip, BPF_ANY);
  if (err) {
    fprintf(stderr, "ERR: bpf_map_update_elem() failed: %s\n", strerror(errno));
    goto fail;
  }

  fprintf(stdout, "Successfully loaded BPF program.\n");
  return 0;

fail:
  return -1;
}