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
  char prog_filename[PATH_MAX];
  char map_subdir[PATH_MAX];
  char map_partial_subdir[PATH_MAX];
  struct bpf_object *fastpath_obj;
  const char *fastpath_file = "bpf_thrift_skmsg.o";

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
  len = snprintf(map_partial_subdir, subdir_len, "%s", cfg.cgroup_name);
  if (len < 0) {
    fprintf(stderr, "ERR: constructing full mapname path\n");
    goto fail;
  }

  // Get fd of the pinned sockops map in pindir/subdir
  len = snprintf(map_filename, PATH_MAX, "%s/%s-%s/%s", pin_dir, map_partial_subdir, "sockops", cfg.map_name);
  if (len < 0) {
    fprintf(stderr, "ERR: constructing full mapname path\n");
    goto fail;
  }

  // Construct the map subdir name for fastpath pin maps.
  len = snprintf(map_subdir, PATH_MAX, "%s-%s", map_partial_subdir, "fastpath");
  if (len < 0) {
    fprintf(stderr, "ERR: creating map dirname\n");
    goto fail;
  }

  len = snprintf(prog_filename, PATH_MAX, "%s/%s-prog/%s", pin_dir, map_partial_subdir, "prog");
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

  // Open, load and attach sockops_obj if not already attached
  fastpath_obj = bpf_object__open_file(fastpath_file, NULL);
  if (libbpf_get_error(fastpath_obj)) {
    fprintf(stderr, "ERR: opening BPF-OBJ file (%s) (%s)\n", fastpath_file,
            strerror(-libbpf_get_error(fastpath_obj)));
    goto fail;
  }

  struct bpf_program *fastpath_prog = bpf_object__find_program_by_name(fastpath_obj, "parse_thrift_payload");
  if (!fastpath_prog) {
    fprintf(stderr, "ERR: finding program by name\n");
    goto fail;
  }

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

  // Pin the map defined in the fastpath program in the pin directory.
  pin_maps_in_bpf_object(fastpath_obj, map_subdir);
  fprintf(stdout, "Successfully loaded BPF program.\n");
  return 0;

fail:
  return -1;
}