#!/usr/bin/python
"""Stitches multiple result JSONs into a single, monotholitic version."""
import os
import json
import csv
import sys

def get_all(input_path):
    """get all json objects"""
    for root, dirs, files in os.walk(input_path):
        # detect files here
        for item in files:
            path = root + "/" + item
            with open(path, 'r') as f:
                j = json.loads(f.read())
                j['z-meta-file'] = path
                idx = 0
                for item in path.split('/'):
                    j['z-meta-' + str(idx)] = item
                    idx = idx + 1
                yield j

def stitch(path, json_out, csv_out):
    """Stitch together."""
    json_out.write("[");
    first = True
    keys = []
    for obj in get_all(path):
        if not first:
            json_out.write(',')
        first = False
        json_out.write(json.dumps(obj,
                                  sort_keys=True,
                                  indent=4,
                                  separators=(',', ': ')
                                  ))
        for k in obj.keys():
            if k not in keys:
                keys.append(k)
    json_out.write("]")
    writer = csv.DictWriter(csv_out,
                            fieldnames=keys,
                            quoting=csv.QUOTE_NONNUMERIC)
    writer.writeheader()
    for obj in get_all(path):
        writer.writerow(obj)

if __name__ == '__main__':
    if len(sys.argv) != 3:
        print('requires an input dir and output file')
        exit(-1)
    output_name = sys.argv[2]
    with open(output_name + '.json', 'w') as json_file:
        with open(output_name + '.csv', 'w') as csv_file:
            stitch(sys.argv[1], json_file, csv_file)
