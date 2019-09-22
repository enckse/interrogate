package main

import (
	"flag"
	"fmt"
	"os"

	"voidedtech.com/survey/core"
)

func main() {
	manifest := flag.String("manifest", "", "manifest file")
	dir := flag.String("dir", "", "directory to use")
	cfg := flag.String("config", "", "configuration file")
	out := flag.String("out", "", "output file naming (prefix)")
	flag.Parse()
	in := core.Inputs{
		Manifest:  *manifest,
		Config:    *cfg,
		Directory: *dir,
		OutName:   *out,
	}
	if err := in.Process(); err != nil {
		fmt.Println(fmt.Sprintf("ERROR: %v", err))
		os.Exit(1)
	}
}
