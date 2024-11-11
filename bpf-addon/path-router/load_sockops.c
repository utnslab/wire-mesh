static const char *__doc__ = "User space program to load the sockops bpf program to register sockets\n";

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

    {{"cgroup", required_argument, NULL, 'c'},
     "Cgroup file to attach to (default: '' => /sys/fs/cgroup/unified)"},

    {{"unload", no_argument, NULL, 'U'},
     "Unload sockops program instead of loading"},

    {{0, 0, NULL, 0}, NULL, false}};

const char *cgroup_dir = "/sys/fs/cgroup/unified";

int main(int argc, char **argv) {
  int err, len;
  int cgroup_fd;
  int subdir_len;
  char *dot;
  char cgroup_filename[PATH_MAX];
  char map_subdir[PATH_MAX];
  char pod_identifier[PATH_MAX];
  const char *sockops_file = "bpf_sockops.o";
  struct bpf_object *sockhash_obj;

  struct config cfg = {
    .cgroup_name = "",
    .do_unload = false,
  };
  parse_cmdline_args(argc, argv, long_options, &cfg, __doc__);

  // Open the cgroup fd -- needed for both attach and detach operations.
  len = snprintf(cgroup_filename, PATH_MAX, "%s/%s", cgroup_dir, cfg.cgroup_name);
  if (len < 0) {
    fprintf(stderr, "ERR: creating cgroup filename\n");
    goto fail;
  }

  cgroup_fd = open(cgroup_filename, O_RDONLY);
  if (cgroup_fd < 0) {
    fprintf(stderr, "ERR: opening cgroup file %s\n", strerror(errno));
    goto exit_cgroup;
  }

  // Check if the program is to be unloaded.
  if (cfg.do_unload) {
    // Unload the sockops program
    err = bpf_prog_detach(cgroup_fd, BPF_CGROUP_SOCK_OPS);
    if (err) {
      fprintf(stderr, "ERR: bpf_prog_detach() failed: %s\n", strerror(errno));
      goto fail;
    }

    printf("Successfully unloaded sockops program from cgroup %s. Please manually remove any loaded maps.\n",
           cfg.cgroup_name);
    return 0;
  }

  // Open, load and attach sockops_obj if not already attached
  sockhash_obj = bpf_object__open_file(sockops_file, NULL);
  if (libbpf_get_error(sockhash_obj)) {
    fprintf(stderr, "ERR: opening BPF-OBJ file (%s) (%s)\n", sockops_file,
            strerror(-libbpf_get_error(sockhash_obj)));
    goto fail;
  }

  struct bpf_program *sockops_prog = bpf_object__find_program_by_name(sockhash_obj, "bpf_add_to_sockhash");
  if (!sockops_prog) {
    fprintf(stderr, "ERR: finding program by name\n");
    goto fail;
  }

  // Load the sockops program
  err = bpf_object__load(sockhash_obj);
  if (err) {
    fprintf(stderr, "ERR: loading BPF-OBJ file %d (%s) (%s)\n", err, sockops_file,
            strerror(-err));
    goto fail;
  }

  // Attach the sockops program
  // Using core BPF API as libbpf doesn't support sockops yet.
  err = bpf_prog_attach(bpf_program__fd(sockops_prog), cgroup_fd, BPF_CGROUP_SOCK_OPS, 0);
  if (err) {
    fprintf(stderr, "ERR: attaching program\n");
    goto fail;
  }

  fprintf(stdout, "Successfully loaded BPF program.\n");

  // Remove '.' from cgroup name
  dot = strchr(cfg.cgroup_name, '.');
  subdir_len = dot ? dot - cfg.cgroup_name + 1 : PATH_MAX;

  // Use only the PodUID characters of cgroup name. 
  len = snprintf(pod_identifier, subdir_len, "%s", cfg.cgroup_name);
  if (len < 0) {
    fprintf(stderr, "ERR: creating map dirname\n");
    goto fail;
  }

  // Pin the map `sock_ops_map` defined in the sockops program in the pin directory.
  len = snprintf(map_subdir, PATH_MAX, "%s-%s", pod_identifier, "sockops");
  if (len < 0) {
    fprintf(stderr, "ERR: creating map dirname\n");
    goto fail;
  }
  pin_maps_in_bpf_object(sockhash_obj, map_subdir);
  return 0;

exit_cgroup:
  close(cgroup_fd);

fail:
  return -1;
}