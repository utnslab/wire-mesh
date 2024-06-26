# Python script to analyze production traces.
# Arguments:
# 1. Directory to the trace files.

import os
import sys
import pickle
from os import listdir
from os.path import isfile, join

path = sys.argv[1]

appls = {}
lines_processed = 0

# Statistics.
counts = {}     # How many times a service is called.

# Read all the files in the directory.
onlyfiles = [f for f in listdir(path) if isfile(join(path, f))]
print(f'Found {len(onlyfiles)} files in the directory.')

# Read each file.
for file in onlyfiles:
    # Skip is the file is gzip.
    if file.endswith('.gz'):
        continue

    print(f'Processing file: {file}')
    with open(os.path.join(path, file), 'r') as f:
        for line in f:
            # Skip the header.
            if lines_processed == 0:
                lines_processed += 1
                continue

            # Read the line, and extract service, UM and DM.
            parts = line.strip().split(',')
            try:
                service = parts[2]
                um = parts[5]
                dm = parts[8]
            except:
                print(f'Error in line: {line}')
                continue

            # Add the edge (um->dm) to the dictionary at service.
            if service not in appls:
                appls[service] = {
                    um: {
                        'count': 0,
                        'in_edges': set(),
                        'out_edges': {dm}
                    },
                    dm: {
                        'count': 1,
                        'in_edges': {um},
                        'out_edges': set()
                    }
                }
            else:
                # Add or update entry for um.
                if um not in appls[service]:
                    appls[service][um] = {
                        'count': 0,
                        'in_edges': set(),
                        'out_edges': {dm}
                    }
                else:
                    appls[service][um]['out_edges'].add(dm)
                
                # Add or update entry for dm.
                if dm not in appls[service]:
                    appls[service][dm] = {
                        'count': 1,
                        'in_edges': {um},
                        'out_edges': set()
                    }
                else:
                    appls[service][dm]['count'] += 1
                    appls[service][dm]['in_edges'].add(um)

            if service not in counts:
                counts[service] = 1
            else:
                counts[service] += 1

            lines_processed += 1
            if lines_processed % 1000000 == 0:
                # Print statistics on how many services found so far.
                # Find the number of services that are called more than 72000 times (10 rps).
                num_services = 0
                for service, count in counts.items():
                    if count > 72000:
                        num_services += 1
                print(f'Found {len(appls)} applications, {num_services} relevant.')

# Conclusion: Almost 99% services are called less than 1000 times.
# Use only the services that are called more than 100 times.
# 
# # Invert the dictionary to get the number of times a service is called.
# inv_counts = {100: 0, 500: 0, 1000: 0, 10000: 0}
# for service, count in counts.items():
#     if count < 100:
#         inv_counts[100] += 1
#     elif count < 500:
#         inv_counts[500] += 1
#     elif count < 1000:
#         inv_counts[1000] += 1
#     else:
#         inv_counts[10000] += 1
# print(f'Number of times a service is called: {inv_counts}')
                        
# Statistics.
num_uservices = {}      # Number of unique microservices.
num_downstream = {}     # Number of downstream microservices.
new_counts = {}         # How many times a service is called.
new_appls = {}          # New dictionary with only relevant services.

for service, uservices in appls.items():
    # Currently -- Do not analyze services that are called less than 72000 times.
    if counts[service] < 72000:
        continue

    new_appls[service] = uservices
    new_counts[service] = counts[service]

    num = len(uservices)
    if num not in num_uservices:
        num_uservices[num] = 1
    else:
        num_uservices[num] += 1
    
    for _, dict in uservices.items():
        num = len(dict['out_edges']) + len(dict['in_edges'])
        if num not in num_downstream:
            num_downstream[num] = 1
        else:
            num_downstream[num] += 1

# Write to a pickle file.
with open('appls.pkl', 'wb') as f:
    pickle.dump(new_appls, f)

with open('counts.pkl', 'wb') as f:
    pickle.dump(new_counts, f)

print(f'Number of unique microservices: {num_uservices}')
print(f'Number of outgoing edges: {num_downstream}')

