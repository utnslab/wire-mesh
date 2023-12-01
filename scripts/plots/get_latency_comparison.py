"""
Plot median and tail latencies to compare the latencies between plain, istio and nginx
for multiple applications.
"""

import os
import sys
import numpy as np

from plot_wrk_comp import get_latencies

if len(sys.argv) < 2:
    print("Usage: python3 get_latency_comparison.py <path> <low/high>")
    sys.exit(1)


path = sys.argv[1]
LOWLOAD = sys.argv[2] == 'low'

applications = ['bookinfo', 'boutique', 'reservation']

# Plot percentiles
pct_to_plot = ['50', '99']

for appl in applications:
    appl_dir = appl
    if LOWLOAD:
        appl_dir += '-lowload'

    with open(os.path.join(path, appl_dir,
                           'time_{0}_plain.run'.format(appl))) as f:
        content = f.readlines()
    content_plain = [x.strip().split() for x in content]
    latencies_plain = get_latencies(content_plain)

    with open(os.path.join(path, appl_dir,
                           'time_{0}_nginx.run'.format(appl))) as f:
        content = f.readlines()
    content_nginx = [x.strip().split() for x in content]
    latencies_nginx = get_latencies(content_nginx)

    with open(os.path.join(path, appl_dir,
                           'time_{0}_istio.run'.format(appl))) as f:
        content = f.readlines()
    content_istio = [x.strip().split() for x in content]
    latencies_istio = get_latencies(content_istio)

    for p in pct_to_plot:
        print("Application: {0} {1}".format(appl, p))
        print("Nginx Latency: {:.2f}".format(latencies_nginx[p] / latencies_plain[p]))
        print("Istio Latency: {:.2f}".format(latencies_istio[p] / latencies_plain[p]))
