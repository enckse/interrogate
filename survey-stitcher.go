package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"voidedtech.com/survey/core"
)

type inputs struct {
	manifest  string
	config    string
	directory string
	outName   string
}

// StitchResult represents a json-ish way of seein gresults
type StitchResult struct {
	Objects []*StitchObject `json:"results"`
}

// StitchObject represents data results
type StitchObject struct {
	File      string `json:"file"`
	client    string
	mode      string
	results   *core.ResultData
	Responses []Response `json:"responses"`
}

// Response is a resulting question/answer from a survey
type Response struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

type fieldData struct {
	core.ExportField
	values []string
	valid  bool
	index  int
}

func (f *fieldData) display() string {
	return fmt.Sprintf("%d. %s (%s)", f.index, f.Text, f.Type)
}

func (i inputs) build(index int, m *core.Manifest, cfg *core.Exports) (*StitchObject, error) {
	o := &StitchObject{
		File:   m.Files[index],
		client: m.Clients[index],
		mode:   m.Modes[index],
	}
	p := filepath.Join(i.directory, fmt.Sprintf("%s.json", o.File))
	if !core.PathExists(p) {
		return nil, fmt.Errorf("invalid manifest file request %s", p)
	}
	b, err := ioutil.ReadFile(p)
	if err != nil {
		return nil, err
	}
	r := &core.ResultData{}
	err = json.Unmarshal(b, &r)
	if err != nil {
		return nil, err
	}
	o.results = r
	var fieldNames []string
	responses := make(map[string]*fieldData)
	actualMode := []string{fmt.Sprintf("mode:%s", o.mode)}
	for k, v := range r.Datum {
		switch k {
		case core.ClientKey:
			continue
		case core.SessionKey, core.TimestampKey:
			actualMode = append(actualMode, fmt.Sprintf("%s:%v", k, v))
			continue
		}
		i, err := strconv.Atoi(k)
		if err != nil {
			return nil, err
		}
		data := &fieldData{
			values: v,
			valid:  false,
			index:  i,
		}
		for cfgIdx, obj := range cfg.Fields {
			if cfgIdx == i {
				data.Text = obj.Text
				data.Type = obj.Type
				data.valid = true
			}
		}
		if !data.valid {
			return nil, fmt.Errorf("invalid data detected: %s %s", k, p)
		}
		disp := data.display()
		fieldNames = append(fieldNames, disp)
		responses[disp] = data
	}
	if len(fieldNames) == 0 {
		return nil, fmt.Errorf("no fields found")
	}
	fieldNames = append(fieldNames, core.ClientKey, core.ModeKey)
	sort.Strings(actualMode)
	o.mode = strings.Join(actualMode, " - ")
	sort.Strings(fieldNames)
	responses[core.ClientKey] = &fieldData{values: []string{o.client}}
	responses[core.ModeKey] = &fieldData{values: []string{o.mode}}
	for _, f := range fieldNames {
		data := "[no response]"
		var useData []string
		for _, val := range responses[f].values {
			if strings.TrimSpace(val) == "" {
				continue
			}
			useData = append(useData, val)
		}
		if len(useData) > 0 {
			data = strings.Join(useData, "\n")
		}
		o.Responses = append(o.Responses, Response{
			Question: f,
			Answer:   data,
		})
	}
	return o, nil
}

func (i inputs) save(results StitchResult) error {
	b, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(fmt.Sprintf("%s.json", i.outName), b, 0644)
	if err != nil {
		return err
	}
	csvFile, err := os.Create(fmt.Sprintf("%s.csv", i.outName))
	if err != nil {
		return err
	}
	defer csvFile.Close()
	var records [][]string
	var header []string
	for idx, obj := range results.Objects {
		var responses []string
		for _, resp := range obj.Responses {
			if idx == 0 {
				header = append(header, resp.Question)
			}
			responses = append(responses, resp.Answer)
		}
		if len(header) > 0 {
			records = append(records, header)
			header = []string{}
		}
		records = append(records, responses)
	}
	fmt.Println(fmt.Sprintf("%v", records))
	writer := csv.NewWriter(csvFile)
	defer writer.Flush()
	err = writer.WriteAll(records)
	if err != nil {
		return err
	}
	return nil
}

func (i inputs) process() error {
	for _, p := range []string{i.manifest, i.config, i.directory} {
		if !core.PathExists(p) {
			return fmt.Errorf("missing required argument")
		}
	}
	if len(i.outName) == 0 {
		return fmt.Errorf("invalid output name information")
	}
	b, err := ioutil.ReadFile(i.manifest)
	if err != nil {
		return err
	}
	m := &core.Manifest{}
	err = json.Unmarshal(b, &m)
	if err != nil {
		return err
	}
	err = m.Check()
	if err != nil {
		return err
	}
	b, err = ioutil.ReadFile(i.config)
	if err != nil {
		return err
	}
	cfg := &core.Exports{}
	err = json.Unmarshal(b, &cfg)
	if err != nil {
		return err
	}
	clients := make(map[string]*StitchObject)
	var clientNames []string
	for idx := range m.Files {
		o, err := i.build(idx, m, cfg)
		if err != nil {
			return err
		}
		clients[o.client] = o
		clientNames = append(clientNames, o.client)
	}
	if len(clientNames) == 0 {
		return fmt.Errorf("no objects found")
	}
	sort.Strings(clientNames)
	overall := StitchResult{}
	for _, name := range clientNames {
		o, _ := clients[name]
		overall.Objects = append(overall.Objects, o)
	}
	return i.save(overall)
}

func main() {
	manifest := flag.String("manifest", "", "manifest file")
	dir := flag.String("dir", "", "directory to use")
	cfg := flag.String("config", "", "configuration file")
	out := flag.String("out", "", "output file naming (prefix)")
	flag.Parse()
	in := inputs{
		manifest:  *manifest,
		config:    *cfg,
		directory: *dir,
		outName:   *out,
	}
	if err := in.process(); err != nil {
		fmt.Println(fmt.Sprintf("ERROR: %v", err))
		os.Exit(1)
	}
}
