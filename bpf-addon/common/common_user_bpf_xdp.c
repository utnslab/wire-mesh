#include <bpf/libbpf.h> /* bpf_get_link_xdp_id + bpf_set_link_xdp_id */
#include <errno.h>
#include <net/if.h>     /* IF_NAMESIZE */
#include <stdlib.h>     /* exit(3) */
#include <string.h>     /* strerror */
#include <sys/stat.h>
#include <unistd.h>

#include <bpf/bpf.h>
#include <bpf/libbpf.h>

#include <linux/if_link.h> /* Need XDP flags */
#include <linux/err.h>

#include "common_defines.h"

#ifndef PATH_MAX
#define PATH_MAX	4096
#endif

#define XDP_UNKNOWN	XDP_REDIRECT + 1
#ifndef XDP_ACTION_MAX
#define XDP_ACTION_MAX (XDP_UNKNOWN + 1)
#endif

static const char *xdp_action_names[XDP_ACTION_MAX] = {
	[XDP_ABORTED]   = "XDP_ABORTED",
	[XDP_DROP]      = "XDP_DROP",
	[XDP_PASS]      = "XDP_PASS",
	[XDP_TX]        = "XDP_TX",
	[XDP_REDIRECT]  = "XDP_REDIRECT",
	[XDP_UNKNOWN]	= "XDP_UNKNOWN",
};

const char *action2str(__u32 action)
{
        if (action < XDP_ACTION_MAX)
                return xdp_action_names[action];
        return NULL;
}

int check_map_fd_info(const struct bpf_map_info *info,
		      const struct bpf_map_info *exp)
{
	if (exp->key_size && exp->key_size != info->key_size) {
		fprintf(stderr, "ERR: %s() "
			"Map key size(%d) mismatch expected size(%d)\n",
			__func__, info->key_size, exp->key_size);
		return EXIT_FAIL;
	}
	if (exp->value_size && exp->value_size != info->value_size) {
		fprintf(stderr, "ERR: %s() "
			"Map value size(%d) mismatch expected size(%d)\n",
			__func__, info->value_size, exp->value_size);
		return EXIT_FAIL;
	}
	if (exp->max_entries && exp->max_entries != info->max_entries) {
		fprintf(stderr, "ERR: %s() "
			"Map max_entries(%d) mismatch expected size(%d)\n",
			__func__, info->max_entries, exp->max_entries);
		return EXIT_FAIL;
	}
	if (exp->type && exp->type  != info->type) {
		fprintf(stderr, "ERR: %s() "
			"Map type(%d) mismatch expected type(%d)\n",
			__func__, info->type, exp->type);
		return EXIT_FAIL;
	}

	return 0;
}

int open_bpf_map_file(const char *pin_dir,
		      const char *mapname,
		      struct bpf_map_info *info)
{
	char filename[PATH_MAX];
	int err, len, fd;
	__u32 info_len = sizeof(*info);

	len = snprintf(filename, PATH_MAX, "%s/%s", pin_dir, mapname);
	if (len < 0) {
		fprintf(stderr, "ERR: constructing full mapname path\n");
		return -1;
	}

	fd = bpf_obj_get(filename);
	if (fd < 0) {
		fprintf(stderr,
			"WARN: Failed to open bpf map file:%s err(%d):%s\n",
			filename, errno, strerror(errno));
		return fd;
	}

	if (info) {
		err = bpf_obj_get_info_by_fd(fd, info, &info_len);
		if (err) {
			fprintf(stderr, "ERR: %s() can't get info - %s\n",
				__func__,  strerror(errno));
			return EXIT_FAIL_BPF;
		}
	}

	return fd;
}

/* Pinning maps under /sys/fs/bpf in subdir */
int pin_maps_in_bpf_object(struct bpf_object *bpf_obj, const char* subdir) {
  char pin_dir[PATH_MAX];
  struct stat pin_dir_stat;
  int err, len;

  len = snprintf(pin_dir, PATH_MAX, "/sys/fs/bpf/%s", subdir);
  if (len < 0) {
	fprintf(stderr, "ERR: creating pin dirname\n");
    return EXIT_FAIL_OPTION;
  }

  /* Existing/previous BPF prog might not have cleaned up */
  if (stat(pin_dir, &pin_dir_stat) == 0 && S_ISDIR(pin_dir_stat.st_mode)) {
    /* Basically calls unlink(3) on map_filename */
    fprintf(stdout, "Pin directory %s exists. Unpinning.\n", pin_dir);
    err = bpf_object__unpin_maps(bpf_obj, pin_dir);
    if (err) {
      fprintf(stderr, "ERR: UNpinning maps in %s\n", pin_dir);
      return EXIT_FAIL_BPF;
    }
  }

  /* This will pin all maps in our bpf_object */
  err = bpf_object__pin_maps(bpf_obj, pin_dir);
  if (err) {
    fprintf(stderr, "ERR: Failed to pin maps.\n");
    return EXIT_FAIL_BPF;
  }

  fprintf(stdout, "Pinning maps succeeded.\n");
  return 0;
}
