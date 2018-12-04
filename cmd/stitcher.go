package main

import (
	"errors"
	"flag"
	"io/ioutil"

	"github.com/epiphyte/goutils/logger"
	"github.com/epiphyte/goutils/opsys"
)

func mergeManifests(files []string, workingFile string) (string, error) {
	if len(files) == 0 {
		return "", errors.New("invalid manifest set")
	}
	if len(files) == 1 {
		return files[0], nil
	}
	clients := make(map[string]string)
	modes := make(map[string]string)
	for _, manifest := range files {
		if opsys.PathNotExists(manifest) {
			return "", errors.New("invalid manifest file given for merge")
		}
		b, err := ioutil.ReadFile(manifest)
		if err != nil {
			return "", err
		}
		read, err := readManifest(b)
		if err != nil {
			return "", err
		}
		err = read.Check()
		if err != nil {
			return "", err
		}
		for i, c := range read.Clients {
			clients[c] = read.Files[i]
			modes[c] = read.Modes[i]
		}
	}
	m := &Manifest{}
	for k, v := range clients {
		m.Files = append(m.Files, v)
		m.Clients = append(m.Clients, k)
		m.Modes = append(m.Modes, modes[k])
	}
	writeManifest(m, workingFile)
	return workingFile, nil
}

func main() {
	var manifests strFlagSlice
	var extensions strFlagSlice
	flag.Var(&manifests, "manifest", "input manifest files")
	dir := flag.String("directory", defaultStore, "location of files to stitch")
	flag.Var(&extensions, "extension", "file extension/type to out (none == all)")
	out := flag.String("output", "results", "output results")
	force := flag.Bool("force", false, "force overwrite existing results")
	flag.Parse()
	logger.WriteInfo(vers)
	manifest, err := mergeManifests(manifests, *out+".manifest")
	if err != nil {
		logger.WriteError("unable to get a unique manifest", err)
		return
	}
	file := manifest
	if opsys.PathNotExists(file) {
		logger.WriteWarn("manifest file not found", file)
		return
	}
	b, err := ioutil.ReadFile(file)
	if err != nil {
		logger.WriteError("unable to read manifest", err)
		return
	}
	m, err := readManifest(b)
	if err != nil {
		logger.WriteError("invalid manifest", err)
		return
	}
	if len(extensions) == 0 {
		extensions = []string{JsonFile, MarkdownFile, CsvFile}
	}
	for _, e := range extensions {
		if e != JsonFile && e != MarkdownFile && e != CsvFile {
			logger.WriteWarn("unknown input extension", e)
			return
		}
		outFile := *out + e
		e := stitch(m, e, *dir, outFile, *force)
		if e != nil {
			logger.WriteError("stitching failed", e)
		}
	}
}
