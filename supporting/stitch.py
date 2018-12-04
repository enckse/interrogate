"""
Supports taking stitched results and producing other outputs.

supported outputs: md, csv
"""
import json
import sys
import csv

SYS = ["client", "session"]


def main():
    """Main entry."""
    if len(sys.argv) < 3:
        print("<cmd> [config] [results.json]")
        exit(1)
    run(sys.argv[1], sys.argv[2])


def run(cfgFile, resultFile): 
    """Run the converter."""
    cfg = {}
    with open(cfgFile) as f:
        cfg = json.loads(f.read())

    results = {}
    with open(resultFile) as f:
        results = json.loads(f.read())

    questions = [(x["text"], x["type"]) for x in cfg["questions"]]
    with open(resultFile + ".md") as f:
        with open(resultFile + ".csv") as c:
            process(questions, f, c)

def process(questions, mdFile, csvFile):
    writer = csv.DictWriter(csvFile, fieldnames=[x[0] for x in questions])
    writer.writeheader()
    for r in results:
        mdFile.write("---")
        obj = r['data']
        idx = {}
        for v in obj:
            values = obj[v]
            if v in SYS:
                continue
            idx[int(v)] = values
        client = r["client"]
        mdFile.write("# {}\n".format(client))
        row = {}
        for k in sorted(i.keys()):
            name = questions[k][0]
            typed = questions[k][1]
            mdFile.write("#### {}. {} ({})\n".format(k, name, typed))
            mdFile.write("```")
            values = idx[k]
            has = False
            resp = ""
            for v in values:
                if v.strip() == "":
                    continue
                mdFile.write(v)
                has = True
                resp = "{}{}".format(resp, v)
            if not has:
                resp = "<no response>"
                mdFile.write(resp)
            row[name] = resp
            mdFile.write("```\n")
        writer.writerow(row)

if __name__ == "__main__":
    main()    
