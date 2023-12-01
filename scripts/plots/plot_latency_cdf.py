"""
Plot a cdf plot comparing the latency of using different service meshes.
"""
import os
import sys
import json
import numpy as np
import matplotlib.pyplot as plt
import matplotlib.patches as mpatches

def get_percentile_data(file):
    with open(file) as f:
        content = f.readlines()
    content = [x.strip().split() for x in content]

    found = False
    cdf = []
    for line in content:
        if not found:
            if len(line) > 0 and line[0] == 'Value':
                found = True
            continue

        if len(line) != 4:
            continue
        cdf.append([float(line[0]), float(line[1])])        
        if line[0][0] == '#':
            break
    
    return cdf


num_applications = 3
applications = ['bookinfo', 'boutique', 'reservation']
directory = sys.argv[1]
LOWLOAD = sys.argv[2] == 'low'

# Read eval_config.json to get directories for each application
current_dir = os.path.dirname(os.path.realpath(__file__))
json_file = os.path.join(current_dir, 'eval_config.json')

with open(json_file) as f:
    results_data = json.load(f)

# Read the files
latencies = []
for appl in applications:
    # Get the directory for the application
    appl_dir = results_data[appl]['heavy']
    if LOWLOAD:
        appl_dir = results_data[appl]['light']

    # Get the percentile data
    plain_file = os.path.join(directory, appl_dir['plain'], 'time_{0}_plain.run'.format(appl))
    istio_file = os.path.join(directory, appl_dir['istio'], 'time_{0}_istio.run'.format(appl))
    wire_file = os.path.join(directory, appl_dir['wire'], 'time_{0}_wire.run'.format(appl))

    latencies_plain = get_percentile_data(plain_file)
    latencies_istio = get_percentile_data(istio_file)
    latencies_wire = get_percentile_data(wire_file)

    latencies.append([latencies_plain, latencies_wire, latencies_istio])

# Make a plot with x-axis as latency and y-axis as percentage of requests
# with latency less than x-axis value. Use the latencies array to make one
# curve each for [plain, istio, wire]. 
# Finally, the plot should have three subplots, one for each application.
# The plot should be saved as a pdf file in the same directory as this script.
# The plot should have a legend and axis labels.
# The plot should have a title that says "CDF of latency for <application>".

# Plot formatting
plt.rcParams['text.usetex'] = True  #Let TeX do the typsetting
plt.rcParams['font.size'] = 14
plt.rcParams['text.latex.preamble'] = [
    r'\usepackage{sansmath}', r'\sansmath'
]  #Force sans-serif math mode (for axes labels)
plt.rcParams['font.family'] = 'sans-serif'  # ... for regular text

fig, axs = plt.subplots(1, num_applications, figsize=(12, 3.5), sharey=True)
for i in range(num_applications):
    # axs[i].plot([x[0] for x in latencies[i][0]], [x[1]*100 for x in latencies[i][0]], color='black', label='Plain')
    axs[i].plot([x[0] for x in latencies[i][1]], [x[1]*100 for x in latencies[i][1]], color='red', label='Wire')
    axs[i].plot([x[0] for x in latencies[i][2]], [x[1]*100 for x in latencies[i][2]], color='blue', label='Istio')

    axs[i].set_xscale('log')
    axs[i].set_ylim(1, 100)
    axs[i].set_xlabel('Latency (ms)')
    if i == 0:
        axs[i].set_ylabel('Percentage of requests')
    axs[i].set_title('CDF of latency for {0}'.format(applications[i]))

    axs[i].legend(loc='lower right')

plt.tight_layout()
plt.savefig('latency_cdf.pdf')
plt.savefig('latency_cdf.png')
