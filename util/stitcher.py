#!/usr/bin/python
"""Stitches multiple result JSONs into a single, monotholitic version."""
import os
import json
import csv
import sys
import argparse

def get_all(input_path, include):
    """get all json objects"""
    for root, dirs, files in os.walk(input_path):
        # detect files here
        for item in files:
            path = root + "/" + item
            if len(include) > 0:
                do_continue = True
                for inc in include:
                    if inc in path:
                        do_continue = False
                        break
                if do_continue:
                    continue
            with open(path, 'r') as f:
                j = json.loads(f.read())
                j['z-meta-file'] = path
                idx = 0
                for item in path.split('/'):
                    j['z-meta-' + str(idx)] = item
                    idx = idx + 1
                yield j

def stitch(path, json_out, csv_out, include):
    """Stitch together."""
    json_out.write("[");
    first = True
    keys = []
    writer = csv.DictWriter(csv_out,
                            fieldnames=keys,
                            quoting=csv.QUOTE_NONNUMERIC)
    writer.writeheader()
    for obj in get_all(path, include):
        writer.writerow(obj)
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

def main():
    """Main entry."""
    parser = argparse.ArgumentParser(description="Survey results handler")
    parser.add_argument("--tag", help="tag to restrict to", required=True)
    parser.add_argument("--store",
                        default="/var/db/survey/",
                        help="data store")
    parser.add_argument("--include",
                        nargs='+',
                        type=str,
                        help="include files")
    parser.add_argument("--output", required=True, help="output file name")
    args = parser.parse_args()
    with open(output_name + '.json', 'w') as json_file:
        with open(output_name + '.csv', 'w') as csv_file:
            stitch(args.store,
                   args.output + ".json",
                   args.output + ".csv",
                   args.include)

if __name__ == "__main__":
    main()
