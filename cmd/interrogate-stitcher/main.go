package main

import (
	"flag"

	"voidedtech.com/interrogate/internal"
)

func main() {
	manifest := flag.String("manifest", "", "manifest file")
	dir := flag.String("dir", "", "directory to use")
	cfg := flag.String("config", "", "configuration file")
	out := flag.String("out", "", "output file naming (prefix)")
	flag.Parse()
	in := internal.Inputs{
		Manifest:  *manifest,
		Config:    *cfg,
		Directory: *dir,
		OutName:   *out,
	}
	if err := in.Process(); err != nil {
		internal.Fatal("processing failure", err)
	}
}
