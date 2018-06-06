package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/epiphyte/goutils"
)

var vers = "master"

func write(b *bytes.Buffer, text string) {
	b.Write([]byte(text))
}

func stitch(m *Manifest, ext, dir, out string) error {
	err := m.Check()
	if err != nil {
		return err
	}
	b := &bytes.Buffer{}
	isJson := ext == JsonFile
	isMarkdown := ext == MarkdownFile
	if isJson {
		write(b, "[\n")
	}
	for i, f := range m.Files {
		client := m.Clients[i]
		mode := m.Modes[i]
		if isJson {
			if i > 0 {
				write(b, "\n,\n")
			}
			write(b, fmt.Sprintf("{\"mode\": \"%s\", \"client\": \"%s\", \"data\":", mode, client))
		}
		if isMarkdown {
			write(b, "---\n")
			write(b, fmt.Sprintf("%s (%s)", client, mode))
			write(b, "\n---\n\n")
		}
		goutils.WriteInfo("stitching client", client)
		path := filepath.Join(dir, f+ext)
		if goutils.PathNotExists(path) {
			return errors.New("missing file for client")
		}
		existing, rerr := ioutil.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		b.Write(existing)
		write(b, "\n")
		if isJson {
			write(b, "}\n")
		}
	}
	if isJson {
		write(b, "]")
	}
	return ioutil.WriteFile(out, b.Bytes(), 0644)
}

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
		if goutils.PathNotExists(manifest) {
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
	flag.Var(&manifests, "manifest", "input manifest files")
	dir := flag.String("directory", StoragePath, "location of files to stitch")
	ext := flag.String("extension", JsonFile, "file extension for stitching")
	out := flag.String("output", "results", "output results")
	flag.Parse()
	goutils.WriteInfo(vers)
	extension := *ext
	if extension != JsonFile && extension != MarkdownFile {
		goutils.WriteWarn("unknown input extension", extension)
		return
	}
	manifest, err := mergeManifests(manifests, *out+".manifest")
	if err != nil {
		goutils.WriteError("unable to get a unique manifest", err)
		return
	}
	file := manifest
	if goutils.PathNotExists(file) {
		goutils.WriteWarn("manifest file not found", file)
		return
	}
	outFile := *out + extension
	b, err := ioutil.ReadFile(file)
	if err != nil {
		goutils.WriteError("unable to read manifest", err)
		return
	}
	m, err := readManifest(b)
	if err != nil {
		goutils.WriteError("invalid manifest", err)
		return
	}
	e := stitch(m, extension, *dir, outFile)
	if e != nil {
		goutils.WriteError("stitching failed", e)
	}
}
