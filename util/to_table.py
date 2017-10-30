#!/usr/bin/python
import csv
import sys
from html import escape

def main():
    """Main entry."""
    if len(sys.argv) != 2:
        print("requires input file (csv)")
        exit(1)
    print("<table>")
    skip_idx = []
    with open(sys.argv[1]) as f:
        reader = csv.reader(f)
        for r in reader:
            idx = 0
            print("<tr>")
            for item in r:
                if item.startswith("z-meta"):
                    skip_idx.append(idx)
                if idx not in skip_idx:
                    print("<td>")
                    print(escape(item))
                    print("</td>")
                idx += 1
            print("</tr>")
    print("</table>")

if __name__ == "__main__":
    main()
