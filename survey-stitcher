#!/usr/bin/python3
"""Stitches survey results together."""
import os
import json
import argparse
import csv
import markdown
import subprocess

_CLIENT = "client"
_MODE = "mode"
_HTML_HEADER = "<html><body>"
_HTML_FOOTER = "</body></html>"


class Data(object):
    """Data object."""

    def __init__(self, disp, typed, actual):
        """Initialize the instance."""
        self.typed = typed
        self.disp = disp
        data = []
        for v in actual:
            if v.strip() == "":
                continue
            data.append(v)
        if len(data) == 0:
            self.data = "<no response>"
        else:
            self.data = "\n".join(data)
            self.data = self.data.strip()

    def to_md(self, md_file):
        """Convert to markdown."""
        md_file.write("\n#### {} ({})\n".format(self.disp, self.typed))
        md_file.write("\n```\n")
        md_file.write(self.data)
        md_file.write("\n```\n")


class Result(object):
    """Result object."""

    def __init__(self, mode, client, data):
        """Initialize the instance."""
        self.mode = mode
        self.client = client
        self.data = data

    def to_md(self, md_file):
        """Convert to md file."""
        md_file.write("\n---\n")
        md_file.write("### {}\n".format(self.client))
        md_file.write("\n```\n{}\n```\n".format(self.mode))
        for d in self.data:
            d.to_md(md_file)
        md_file.write("\n")

    def to_object(self, json_file, csv_file):
        """Convert to object-based output."""
        obj = {}
        obj[_CLIENT] = self.client
        obj[_MODE] = self.mode
        for d in self.data:
            obj[d.disp] = d.data
        csv_file.writerow(obj)
        json_file.write(json.dumps(obj))


def display(number, text):
    """Output display text."""
    return "{}. {}".format(number, text)


def set_if_unset(file_name, key, prefix, value, directory):
    """Set a config value if not set."""
    match = False
    if prefix and file_name.startswith(key):
        match = True
    if not prefix and file_name.endswith(key):
        match = True
    if match:
        if value:
            print("found {} but already given ({})".format(key, value))
        else:
            return os.path.join(directory, file_name)
    return value


def autodetect(args):
    """Auto detect settings."""
    if not os.path.exists(args.dir):
        print("directory {} does not exist".format(args.dir))
        return None
    for f in os.listdir(args.dir):
        args.config = set_if_unset(f,
                                   "run.config.",
                                   True,
                                   args.config,
                                   args.dir)
        args.manifest = set_if_unset(f,
                                     ".index.manifest",
                                     False,
                                     args.manifest,
                                     args.dir)
    return args


def main():
    """Program entry point."""
    parser = argparse.ArgumentParser(description="stitch survey results")
    parser.add_argument("--manifest", help="input manifest")
    parser.add_argument("--dir", required=True, help="directory of files")
    parser.add_argument("--config", help="input config")
    parser.add_argument("--out", default="results", help="output file(s)")
    parser.add_argument("--auto",
                        action="store_true",
                        help="auto detect file inputs")
    args = parser.parse_args()
    try:
        if args.auto:
            args = autodetect(args)
            if args is None:
                return
        stop = False
        if args.manifest is None:
            print("manifest is required")
            stop = True
        if args.config is None:
            print("config is required")
            stop = True
        if stop:
            return
        print("processing...")
        run(args)
    except Exception as e:
        print("unable to process for stitching")
        print(e)
        exit(1)


def load_config(config):
    """Load a config to objects."""
    cfg = {}
    with open(config) as f:
        cfg = json.loads(f.read())
    return list([(x["text"], x["type"]) for x in cfg])


def run(args):
    """Run the stitcher."""
    manifest = {}
    with open(args.manifest) as f:
        manifest = json.loads(f.read())
    results = []
    questions = load_config(args.config)
    objs = []
    files = manifest["files"]
    modes = manifest["modes"]
    idx = 0
    print("parsing clients...")
    for client in manifest["clients"]:
        m = modes[idx]
        f = files[idx]
        idx += 1
        obj = {"client": client, "mode": m}
        with open(os.path.join(args.dir, f + ".json")) as f:
            obj["data"] = json.loads(f.read())
        objs.append(obj)
    print("reading data...")
    for mani in objs:
        client = mani["client"]
        mode = [mani["mode"]]
        data = mani['data']
        idx = {}
        for v in data:
            values = data[v]
            if v == "client":
                continue
            if v in ["session", "timestamp"]:
                mode = mode + values
                continue
            idx[int(v)] = values
        datum = []
        for k in sorted(idx.keys()):
            text = questions[k][0]
            typed = questions[k][1]
            disp = display(k, text)
            datum.append(Data(disp, typed, idx[k]))
        obj = Result(" - ".join(mode), client, datum)
        results.append(obj)
    fields = list([display(ind, x[0]) for ind, x in enumerate(questions)])
    fields += [_CLIENT, _MODE]
    fields = list(sorted(fields))
    markdown_file = args.out + ".md"
    print("outputs...")
    tar_files = []
    j_file = args.out + ".json"
    c_file = args.out + ".csv"
    tar_files.append(j_file)
    tar_files.append(c_file)
    tar_files.append(markdown_file)
    with open(j_file, 'w') as j_file:
        with open(markdown_file, 'w') as md_file:
            with open(c_file, 'w') as c_file:
                csv_file = csv.DictWriter(c_file, fieldnames=fields)
                csv_file.writeheader()
                j_file.write("[\n")
                first = True
                for result in results:
                    result.to_md(md_file)
                    if not first:
                        j_file.write("\n,\n")
                    first = False
                    result.to_object(j_file, csv_file)
                j_file.write("\n]")
    h_file = args.out + ".html"
    tar_files.append(h_file)
    with open(h_file, 'w') as h_file:
        h_file.write(_HTML_HEADER)
        with open(markdown_file) as m_file:
            html = markdown.markdown(m_file.read())
            h_file.write(html)
        h_file.write(_HTML_FOOTER)
    tar_files = [os.path.basename(x) for x in tar_files]
    subprocess.call(["tar",
                     "cvzf",
                     args.out + ".tar.gz",
                     "-C", os.path.dirname(args.out)] + tar_files)


if __name__ == "__main__":
    main()
