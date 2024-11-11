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

    {{"unload", no_argument, NULL, 'U'},
     "Unload sockops program instead of loading"},

    {{0, 0, NULL, 0}, NULL, false}};

const char *pin_dir = "/sys/fs/bpf";

int main(int argc, char **argv) {
  int err, len;
  int subdir_len;
  int prog_fd, sock_map_fd;
  char* dot;
  char map_filename[PATH_MAX];
  char parser_prog[PATH_MAX];
  char verdict_prog[PATH_MAX];
  char map_dir[PATH_MAX];
  char pod_identifier[PATH_MAX];
  struct bpf_object *bpf_skb_obj;
  const char *bpf_skb_file = "bpf_sk_skb.o";

  struct config cfg = {
    .cgroup_name = "",
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

  // Construct the map dir name for pin maps.
  len = snprintf(map_dir, PATH_MAX, "%s/%s", pin_dir, pod_identifier);
  if (len < 0) {
    fprintf(stderr, "ERR: creating map dirname\n");
    goto fail;
  }

  len = snprintf(parser_prog, PATH_MAX, "%s/%s-prog/%s", pin_dir, pod_identifier, "parser-prog");
  if (len < 0) {
    fprintf(stderr, "ERR: constructing pin prog path\n");
    goto fail;
  }

  len = snprintf(verdict_prog, PATH_MAX, "%s/%s-prog/%s", pin_dir, pod_identifier, "verdict-prog");
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
    // Get the bpf_skb program fd
    prog_fd = bpf_obj_get(parser_prog);
    if (prog_fd < 0) {
      fprintf(stderr, "ERR: bpf_obj_get(%s): %s\n", parser_prog, strerror(errno));
      goto fail;
    }

    // Detach the bpf_skb program from the map.
    err = bpf_prog_detach2(prog_fd, sock_map_fd, BPF_SK_SKB_STREAM_PARSER);
    if (err) {
      fprintf(stderr, "ERR: bpf_prog_detach2() failed: %s\n", strerror(errno));
      goto fail;
    }

    prog_fd = bpf_obj_get(verdict_prog);
    if (prog_fd < 0) {
      fprintf(stderr, "ERR: bpf_obj_get(%s): %s\n", verdict_prog, strerror(errno));
      goto fail;
    }

    // Detach the bpf_skb program from the map.
    err = bpf_prog_detach2(prog_fd, sock_map_fd, BPF_SK_SKB_STREAM_VERDICT);
    if (err) {
      fprintf(stderr, "ERR: bpf_prog_detach2() failed: %s\n", strerror(errno));
      goto fail;
    }

    printf("Successfully unloaded bpf_skb program via detach2. Please manually remove any loaded maps.\n");
    return 0;
  }

  // Open, load and attach sockops_obj if not already attached
  DECLARE_LIBBPF_OPTS(bpf_object_open_opts, opts,
        .pin_root_path = map_dir);
  bpf_skb_obj = bpf_object__open_file(bpf_skb_file, &opts);
  if (libbpf_get_error(bpf_skb_obj)) {
    fprintf(stderr, "ERR: opening BPF-OBJ file (%s) (%s)\n", bpf_skb_file,
            strerror(-libbpf_get_error(bpf_skb_obj)));
    goto fail;
  }

  // Load the sk_skb program
  err = bpf_object__load(bpf_skb_obj);
  if (err) {
    fprintf(stderr, "ERR: loading BPF-OBJ file for parser (%s)\n", strerror(-err));
    goto fail;
  }

  // Find the stream_parser program
  struct bpf_program *skb_parser_prog = bpf_object__find_program_by_name(bpf_skb_obj, "parse_skb");
  if (!skb_parser_prog) {
    fprintf(stderr, "ERR: finding parser program by name\n");
    goto fail;
  }

  // Find the stream_verdict program
  struct bpf_program *skb_verdict_prog = bpf_object__find_program_by_name(bpf_skb_obj, "read_skb");
  if (!skb_verdict_prog) {
    fprintf(stderr, "ERR: finding verdict program by name\n");
    goto fail;
  }

  // Pin the skb_parser program in the pin directory.
  err = bpf_program__pin(skb_parser_prog, parser_prog);
  if (err) {
    fprintf(stderr, "ERR: bpf_program__pin(%s) failed: %s\n", parser_prog, strerror(errno));
    goto fail;
  }

  // Pin the skb_parser program in the pin directory.
  err = bpf_program__pin(skb_verdict_prog, verdict_prog);
  if (err) {
    fprintf(stderr, "ERR: bpf_program__pin(%s) failed: %s\n", verdict_prog, strerror(errno));
    goto fail;
  }

  // Attach the skb_parser program to the map.
  err = bpf_prog_attach(bpf_program__fd(skb_parser_prog), sock_map_fd, BPF_SK_SKB_STREAM_PARSER, 0);
  if (err) {
    fprintf(stderr, "ERR: bpf_program__attach() failed: %s\n", strerror(errno));
    goto fail;
  }

  err = bpf_prog_attach(bpf_program__fd(skb_verdict_prog), sock_map_fd, BPF_SK_SKB_STREAM_VERDICT, 0);
  if (err) {
    fprintf(stderr, "ERR: bpf_program__attach() failed: %s\n", strerror(errno));
    goto fail;
  }

  // Pin the map defined in the fastpath program in the pin directory.
  // pin_maps_in_bpf_object(bpf_skb_obj, map_subdir);

  fprintf(stdout, "Successfully loaded BPF program.\n");
  return 0;

fail:
  return -1;
}