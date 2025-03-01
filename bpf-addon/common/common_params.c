#include <stddef.h>
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <stdbool.h>
#include <getopt.h>
#include <errno.h>

#include <net/if.h>
#include <linux/if_link.h> /* XDP_FLAGS_* depend on kernel-headers installed */
#include <linux/if_xdp.h>

#include "common_params.h"

int verbose = 1;

#define BUFSIZE 30

void _print_options(const struct option_wrapper *long_options, bool required)
{
	int i, pos;
	char buf[BUFSIZE];

	for (i = 0; long_options[i].option.name != 0; i++) {
		if (long_options[i].required != required)
			continue;

		if (long_options[i].option.val > 64) /* ord('A') = 65 */
			printf(" -%c,", long_options[i].option.val);
		else
			printf("    ");
		pos = snprintf(buf, BUFSIZE, " --%s", long_options[i].option.name);
		if (long_options[i].metavar)
			snprintf(&buf[pos], BUFSIZE-pos, " %s", long_options[i].metavar);
		printf("%-22s", buf);
		printf("  %s", long_options[i].help);
		printf("\n");
	}
}

void usage(const char *prog_name, const char *doc,
           const struct option_wrapper *long_options, bool full)
{
	printf("Usage: %s [options]\n", prog_name);

	if (!full) {
		printf("Use --help (or -h) to see full option list.\n");
		return;
	}

	printf("\nDOCUMENTATION:\n %s\n", doc);
	printf("Required options:\n");
	_print_options(long_options, true);
	printf("\n");
	printf("Other options:\n");
	_print_options(long_options, false);
	printf("\n");
}

int option_wrappers_to_options(const struct option_wrapper *wrapper,
				struct option **options)
{
	int i, num;
	struct option *new_options;
	for (i = 0; wrapper[i].option.name != 0; i++) {}
	num = i;

	new_options = malloc(sizeof(struct option) * num);
	if (!new_options)
		return -1;
	for (i = 0; i < num; i++) {
		memcpy(&new_options[i], &wrapper[i], sizeof(struct option));
	}

	*options = new_options;
	return 0;
}

void parse_cmdline_args(int argc, char **argv,
			const struct option_wrapper *options_wrapper,
                        struct config *cfg, const char *doc)
{
	struct option *long_options;
	bool full_help = false;
	int longindex = 0;
	char *dest;
	int opt;

	if (option_wrappers_to_options(options_wrapper, &long_options)) {
		fprintf(stderr, "Unable to malloc()\n");
		exit(EXIT_FAIL_OPTION);
	}

	/* Parse commands line args */
	while ((opt = getopt_long(argc, argv, "hd:r:R:ANFUqQ:c:m:s:",
				  long_options, &longindex)) != -1) {
		switch (opt) {
		case 'd':
			if (strlen(optarg) > IF_NAMESIZE) {
				fprintf(stderr, "ERR: --dev name too long %ld.\n", strlen(optarg));
				goto error;
			}
			cfg->ifname = (char *)&cfg->ifname_buf;
			strncpy(cfg->ifname, optarg, IF_NAMESIZE);
			cfg->ifindex = if_nametoindex(cfg->ifname);
			if (cfg->ifindex == 0) {
				fprintf(stderr,
					"ERR: --dev name unknown err(%d):%s\n",
					errno, strerror(errno));
				goto error;
			}
			break;
		case 'r':
		case 'R':
			if (strlen(optarg) >= IF_NAMESIZE) {
				fprintf(stderr, "ERR: --redirect-dev name too long\n");
				goto error;
			}
			cfg->redirect_ifname = (char *)&cfg->redirect_ifname_buf;
			strncpy(cfg->redirect_ifname, optarg, IF_NAMESIZE);
			cfg->redirect_ifindex = if_nametoindex(cfg->redirect_ifname);
			if (cfg->redirect_ifindex == 0) {
				fprintf(stderr,
						"ERR: --redirect-dev name unknown err(%d):%s\n",
						errno, strerror(errno));
				goto error;
			}
			break;
		case 'A':
			cfg->xdp_flags &= ~XDP_FLAGS_MODES;    /* Clear flags */
			break;
		case 'N':
			cfg->xdp_flags &= ~XDP_FLAGS_MODES;    /* Clear flags */
			cfg->xdp_flags |= XDP_FLAGS_DRV_MODE;  /* Set   flag */
			break;
		case 'F':
			cfg->xdp_flags &= ~XDP_FLAGS_UPDATE_IF_NOEXIST;
			break;
		case 'U':
			cfg->do_unload = true;
			break;
		case 'q':
		case 'Q':
			verbose = false;
			break;
		case 'c':
			dest  = (char *)&cfg->cgroup_name;
			strncpy(dest, optarg, sizeof(cfg->cgroup_name));
			break;
		case 'm':
			dest  = (char *)&cfg->map_name;
			strncpy(dest, optarg, sizeof(cfg->map_name));
			break;
		case 's':
			dest  = (char *)&cfg->service_ip;
			strncpy(dest, optarg, sizeof(cfg->service_ip));
			break;
		case 1: /* --filename */
			dest  = (char *)&cfg->filename;
			strncpy(dest, optarg, sizeof(cfg->filename));
			break;
		case 2: /* --progsec */
			dest  = (char *)&cfg->progsec;
			strncpy(dest, optarg, sizeof(cfg->progsec));
			break;
		case 3: /* --offload-mode */
			cfg->xdp_flags &= ~XDP_FLAGS_MODES;    /* Clear flags */
			cfg->xdp_flags |= XDP_FLAGS_HW_MODE;   /* Set   flag */
			break;
		case 'h':
			full_help = true;
			/* fall-through */
		error:
		default:
			usage(argv[0], doc, options_wrapper, full_help);
			free(long_options);
			exit(EXIT_FAIL_OPTION);
		}
	}
	free(long_options);
}
