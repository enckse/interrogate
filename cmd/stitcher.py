#!/usr/bin/python
"""Stitches survey results together."""
import json
import argparse
import csv
import markdown

_CLIENT = "client"
_MODE = "mode"


class Data(object):
    """Data object."""

    def __init__(self, text, typed, actual):
        """Initialize the instance."""
        self.text = text
        self.typed = typed
        data = []
        for v in actual:
            if v.strip() == "":
                continue
            data.append(v)
        if len(data) == 0:
            self.data = "<no response>"
        else:
            self.data = "\n".join(data)

    def to_md(self, md_file):
        """Convert to markdown."""
        md_file.write("#### {} ({})".format(self.text, self.typed))
        md_file.write("```")
        md_file.write(self.data)
        md_file.write("```")


class Result(object):
    """Result object."""

    def __init__(self, mode, client, data):
        """Initialize the instance."""
        self.mode = mode
        self.client = client
        self.data = data

    def to_md(self, md_file):
        """Convert to md file."""
        md_file.write("---")
        md_file.write("### {} ({})".format(client, mode))
        for d in self.data:
            d.to_md(md_file)

    def to_csv(self, csv_file):
        """Convert to csv file."""
        obj = {}
        obj[_CLIENT] = self.client
        obj[_MODE] = self.mode
        for d in self.data:
            obj[d.text] = d.data
        csv_file.writerow(obj)


def main():
    """Program entry point."""
    parser = argparse.ArgumentParser(description="stitch survey results")
    parser.add_argument("--manifest", required=True, help="input manifest")
    parser.add_argument("--config", required=True, help="input config")
    parser.add_argument("--dir", required=True, help="directory of files")
    parser.add_argument("--out", default="results", help="output file(s)")
    args = parser.parse_args()
    try:
        run(args)
    except e as Exception:
        print("unable to process for stitching")
        print(e)
        exit(1)


def run(args):
    """Run the stitcher."""
    cfg = {}
    with open(args.config) as f:
        cfg = json.loads(f.read())
    manifest = {}
    with open(args.manifest) as f:
        manifest = json.loads(f.read())
    results = []
    questions = list([(x["text"], x["type"]) for x in cfg["questions"]])
    for mani in manifest:
        client = mani["client"]
        mode = mani["mode"]
        data = mani['data']
        idx = {}
        for v in o:
            values = o[v]
            if v in ["session", "client"]:
                continue
            idx[int(v)] = values
        datum = []
        for k in sorted(idx.keys()):
            text = q[k][0]
            typed = q[k][1]
            datum.append(Data(text, typed, idx[k]))
        obj = Result(mode, client, datum)
        results.append(obj)
    fields = list([x[0] for x in questions])
    markdown_file = args.out + ".md"
    with open(args.out + ".json") as j_file:
        with open(markdown_file) as md_file:
            with open(args.out + ".csv") as c_file:
                csv_file = csv.DictWriter(c_file, fieldnames=fields)
                csv_file.writeheader()
                for result in results:
                    result.to_json(j_file)
                    result.to_md(md_file)
                    result.to_csv(csv_file)
    with open(args.out + ".html") as h_file:
        with open(markdown_file) as m_file:
            html = markdown.markdown(m_file.read())
            h_file.write(html)


if __name__ == "__main__":
    main()
