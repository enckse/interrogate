package main

import (
	"bytes"
	"errors"
	"flag"
	"io/ioutil"
	"path/filepath"

	"github.com/epiphyte/goutils"
)

func stitch(m *Manifest, ext, dir, out string) error {
	if len(m.Clients) != len(m.Files) {
		return errors.New("invalid manifest files!=clients")
	}
	var b bytes.Buffer
	isJson := ext == JsonFile
	isMarkdown := ext == MarkdownFile
	if isJson {
		b.Write([]byte("["))
	}
	for i, f := range m.Files {
		if i > 0 {
			b.Write([]byte(","))
		}
		client := m.Clients[i]
		if isMarkdown {
			b.Write([]byte("---\n"))
			b.Write([]byte(client))
			b.Write([]byte("\n---\n\n"))
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
		b.Write([]byte("\n"))
	}
	if isJson {
		b.Write([]byte("]"))
	}
	return ioutil.WriteFile(out, b.Bytes(), 0644)
}

func main() {
	manifest := flag.String("manifest", "", "input manifest file")
	dir := flag.String("directory", StoragePath, "location of files to stitch")
	ext := flag.String("extension", JsonFile, "file extension for stitching")
	out := flag.String("output", "results", "output results")
	flag.Parse()
	extension := *ext
	if extension != JsonFile && extension != MarkdownFile {
		goutils.WriteWarn("unknown input extension", extension)
		return
	}
	file := *manifest
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
