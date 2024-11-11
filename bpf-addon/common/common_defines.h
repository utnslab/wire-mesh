#ifndef __COMMON_DEFINES_H
#define __COMMON_DEFINES_H

#include <net/if.h>
#include <linux/types.h>
#include <stdbool.h>

struct config {
	__u32 xdp_flags;
	int ifindex;
	char *ifname;
	char ifname_buf[IF_NAMESIZE];
	int redirect_ifindex;
	char *redirect_ifname;
	char redirect_ifname_buf[IF_NAMESIZE];
	bool do_unload;
	char config_file[512];
	char pin_dir[512];
	char filename[512];
	char progsec[32];
	char cgroup_name[512];
	char map_name[512];
	char service_ip[512];
};

#ifndef PATH_MAX
#define PATH_MAX 4096
#endif

/* Defined in common_params.o */
extern int verbose;

/* Exit return codes */
#define EXIT_OK 		 0 /* == EXIT_SUCCESS (stdlib.h) man exit(3) */
#define EXIT_FAIL		 1 /* == EXIT_FAILURE (stdlib.h) man exit(3) */
#define EXIT_FAIL_OPTION	 2
#define EXIT_FAIL_XDP		30
#define EXIT_FAIL_BPF		40

#endif /* __COMMON_DEFINES_H */
