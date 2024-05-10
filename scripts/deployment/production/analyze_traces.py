# Python script to read the pickle files with condensed traces.
# Present analysis of the traces to reveal graph information.

import json
import pickle
import numpy as np
import matplotlib.pyplot as plt

# Read the pickle file.
with open('appls.pkl', 'rb') as f:
    appls = pickle.load(f)

with open('counts.pkl', 'rb') as f:
    counts = pickle.load(f)

# Print the statistics.
print(f'Found {len(appls)} services in the traces.')

num_uservices = {}      # Number of unique microservices.
num_downstream = {}     # Number of downstream microservices.
edge_count_map = {}     # Map number of downstream microservices to the number of times such services are called.
outgoing_edges = []     # List to store the number of outgoing edges.
leaf_node_fraction = [] # List to store the fraction of leaf nodes.

json_data = {}          # JSON data to store the graph information.

for service, uservices in appls.items():
    num_services = len(uservices)
    if num_services not in num_uservices:
        num_uservices[num_services] = 1
    else:
        num_uservices[num_services] += 1

    json_data[service] = {}
    leaf_services = 0
    for svc, meta in uservices.items():
        num = len(meta['out_edges'])
        if num not in num_downstream:
            num_downstream[num] = 1
        else:
            num_downstream[num] += 1

        outgoing_edges.append(num)
        if num == 0:
            leaf_services += 1
        
        # Count the number of times a service is called.
        total_count = meta['count']
        num_edges = len(meta['out_edges']) + len(meta['in_edges'])
        
        if num_edges not in edge_count_map:
            edge_count_map[num_edges] = total_count
        else:
            edge_count_map[num_edges] += total_count
        
        # Store the graph information.
        json_data[service][svc] = {
            'num_edges': num_edges,
            'edges': list(meta['out_edges'])
        }
    
    leaf_node_fraction.append(leaf_services / num_services)

print(f'Number of unique microservices: {num_uservices}')
print(f'Number of outgoing edges: {num_downstream}')
print(f'Number of times a service is called, for the number of edges: {edge_count_map}')

# Find the median number of outgoing edges.
outgoing_edges = np.array(outgoing_edges)
median = np.median(outgoing_edges)
mean = np.mean(outgoing_edges)
print(f'Median number of outgoing edges: {median}')
print(f'Mean number of outgoing edges: {mean}')

# Write the graph information to a JSON file.
with open('appls.json', 'w') as f:
    json.dump(json_data, f, indent=4)

# # Print edge_count_map with keys sorted.
# for key, value in sorted(edge_count_map.items(), key=lambda item: item[0]):
#     print(f'{key}: {value}')

# Plot formatting
plt.rcParams['text.usetex'] = True  #Let TeX do the typsetting
plt.rcParams['font.size'] = 14
plt.rcParams['text.latex.preamble'] = [
    r'\usepackage{sansmath}', r'\sansmath'
]  #Force sans-serif math mode (for axes labels)
plt.rcParams['font.family'] = 'sans-serif'  # ... for regular text
plt.rcParams[
    'font.sans-serif'] = 'Computer Modern Sans serif'  # Choose a nice font here

# # Plot the CDF of the number of outgoing edges.
# outgoing_edges.sort()
# total_count = len(outgoing_edges)
# cdf = 0
# x = []
# y = []
# for num in outgoing_edges:
#     cdf += 1
#     x.append(num)
#     y.append(cdf / total_count)

# plt.plot(x, y)
# plt.xlabel('Number of outgoing edges')
# plt.ylabel('CDF')
# plt.title('CDF of the number of outgoing edges')
# plt.show()

# Plot a CDF of the number of times a service, with a given number of edges, is called.
edge_count_map = dict(sorted(edge_count_map.items(), key=lambda item: item[0]))
total_count = sum(edge_count_map.values())
cdf = 0
x = []
y = []
for key, value in edge_count_map.items():
    cdf += value
    x.append(key)
    y.append(cdf / total_count)

fig = plt.figure(figsize=(7.2, 3.6))
plt.plot(x, y, linewidth=6)

# Draw a horizontal line at y = 0.5 until x = 4.
plt.axhline(y=0.5, color='r', linestyle='--', linewidth=2)

plt.yticks(np.arange(0, 1.1, 0.2))
plt.xscale('log')
plt.xlabel('Number of edges in dependency graph', fontsize=26)
plt.ylabel('Cumulative Fraction\nof Invocations', fontsize=26)

plt.tick_params(axis='both', which='major', labelsize=22)
plt.grid(True, which='both', linestyle='--', linewidth=1)

plt.tight_layout()
plt.subplots_adjust(left=0.2, right=0.95, top=0.9, bottom=0.25)
# plt.show()
plt.savefig('cdf_hotspots.pdf')
plt.savefig('cdf_hotspots.png')
# plt.close()

# Plot a CDF of the fraction of leaf nodes.
leaf_node_fraction.sort()
total_count = len(leaf_node_fraction)
cdf = 0
x = []
y = []
for num in leaf_node_fraction:
    cdf += 1
    x.append(num)
    y.append(cdf / total_count)

fig = plt.figure(figsize=(4.8, 3.6))
plt.plot(x, y, linewidth=6)

# Draw a horizontal line at y = 0.5 until x = 4.
plt.axhline(y=0.5, color='r', linestyle='--', linewidth=2)

plt.yticks(np.arange(0, 1.1, 0.2))
plt.xticks(np.arange(0, 1.1, 0.25))
plt.xlabel('Fraction of leaf nodes', fontsize=26)
plt.ylabel('CDF', fontsize=26)

plt.tick_params(axis='both', which='major', labelsize=24)
plt.grid(True, which='both', linestyle='--', linewidth=1)

plt.tight_layout()
plt.subplots_adjust(left=0.22, right=0.92, top=0.95, bottom=0.25)
# plt.show()
plt.savefig('cdf_leaf_nodes.pdf')
plt.savefig('cdf_leaf_nodes.png')
