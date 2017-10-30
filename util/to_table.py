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
    with open(sys.argv[1]) as f:
        reader = csv.reader(f)
        for r in reader:
            print("<tr>")
            for item in r:
                print("<td>")
                print(escape(item))
                print("</td>")
            print("</tr>")
    print("</table>")

if __name__ == "__main__":
    main()


