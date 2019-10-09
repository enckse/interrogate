package internal

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html"
	"html/template"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const (
	templateHTML = `<!doctype html>
<html lang="en">
<style>
pre{
    white-space: pre-wrap;
}
</style>
<body>
<div>
{{ range $okey, $resp := .Objects }}
{{ if $resp.Start }}<hr />{{ end }}
	<h4>{{ $resp.Question }}</h4>
	<pre>{{ $resp.HTMLResponse }}</pre>
{{ if $resp.End }}<hr />{{ end }}
{{ end }}
</div>
</body>
</html>`
)

type (
	// Inputs represent stitching inputs
	Inputs struct {
		Manifest  string
		Config    string
		Directory string
		OutName   string
	}

	// TemplateResult displays/formats for HTML output
	TemplateResult struct {
		Objects []*TemplateResponse
	}

	// TemplateResponse is an HTML friendly response
	TemplateResponse struct {
		Question     string
		HTMLResponse string
		Start        bool
		End          bool
	}
	// StitchResult represents a json-ish way of seein gresults
	StitchResult struct {
		Objects []*StitchObject `json:"results"`
	}

	// StitchObject represents data results
	StitchObject struct {
		File      string `json:"file"`
		client    string
		mode      string
		results   *ResultData
		Responses []Response `json:"responses"`
	}

	// Response is a resulting question/answer from a survey
	Response struct {
		Question string `json:"question"`
		Answer   string `json:"answer"`
	}

	fieldData struct {
		ExportField
		values []string
		index  int
	}
)

func (s *StitchResult) toHTML(file string) error {
	tmpl, err := template.New("t").Parse(templateHTML)
	if err != nil {
		return err
	}
	obj := &TemplateResult{}
	for _, o := range s.Objects {
		totalResp := len(o.Responses) - 1
		for idx, r := range o.Responses {
			resp := &TemplateResponse{
				Start:        idx == 0,
				End:          idx == totalResp,
				Question:     html.EscapeString(r.Question),
				HTMLResponse: html.EscapeString(r.Answer),
			}
			obj.Objects = append(obj.Objects, resp)
		}
	}
	html, err := os.Create(file)
	if err != nil {
		return err
	}
	defer html.Close()
	return tmpl.Execute(html, obj)
}

func (f *fieldData) display() string {
	return fmt.Sprintf("%02d. %s (%s)", f.index, f.Text, f.Type)
}

func (i Inputs) build(index int, m *Manifest, cfg *Exports) (*StitchObject, error) {
	o := &StitchObject{
		File:   m.Files[index],
		client: m.Clients[index],
		mode:   m.Modes[index],
	}
	p := filepath.Join(i.Directory, fmt.Sprintf("%s.json", o.File))
	if !PathExists(p) {
		return nil, fmt.Errorf("invalid manifest file request %s", p)
	}
	b, err := ioutil.ReadFile(p)
	if err != nil {
		return nil, err
	}
	r := &ResultData{}
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, err
	}
	o.results = r
	var fieldNames []string
	responses := make(map[string]*fieldData)
	var timestamp []string
	var session []string
	actualMode := []string{fmt.Sprintf("mode:%s", o.mode)}
	for cfgIdx, obj := range cfg.Fields {
		data := &fieldData{
			index: cfgIdx,
		}
		data.Text = obj.Text
		data.Type = obj.Type
		for k, v := range r.Datum {
			switch k {
			case ClientKey:
				continue
			case SessionKey:
				session = v
				continue
			case TimestampKey:
				timestamp = v
				continue
			}
			i, err := strconv.Atoi(k)
			if err != nil {
				return nil, err
			}
			if cfgIdx == i {
				data.values = v
			}
		}
		disp := data.display()
		fieldNames = append(fieldNames, disp)
		responses[disp] = data
	}
	actualMode = append(actualMode, fmt.Sprintf("session:%v", session))
	actualMode = append(actualMode, fmt.Sprintf("timestamp:%v", timestamp))
	if len(fieldNames) == 0 {
		return nil, fmt.Errorf("no fields found")
	}
	fieldNames = append(fieldNames, ClientKey, ModeKey)
	sort.Strings(actualMode)
	o.mode = strings.Join(actualMode, " - ")
	sort.Strings(fieldNames)
	responses[ClientKey] = &fieldData{values: []string{o.client}}
	responses[ModeKey] = &fieldData{values: []string{o.mode}}
	for _, f := range fieldNames {
		var useData []string
		data := ""
		for _, val := range responses[f].values {
			data = "[no response]"
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

func (i Inputs) save(results StitchResult) error {
	b, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return err
	}
	jFile := fmt.Sprintf("%s.json", i.OutName)
	hFile := fmt.Sprintf("%s.html", i.OutName)
	cFile := fmt.Sprintf("%s.csv", i.OutName)
	if err := ioutil.WriteFile(jFile, b, 0644); err != nil {
		return err
	}
	if err := results.toHTML(hFile); err != nil {
		return err
	}
	csvFile, err := os.Create(cFile)
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
	writer := csv.NewWriter(csvFile)
	defer writer.Flush()
	if err := writer.WriteAll(records); err != nil {
		return err
	}
	args := []string{"czvf", fmt.Sprintf("%s.tar.gz", filepath.Base(i.OutName))}
	for _, f := range []string{hFile, cFile, jFile} {
		args = append(args, filepath.Base(f))
	}
	cmd := exec.Command("tar", args...)
	cmd.Dir = filepath.Dir(i.OutName)
	return cmd.Run()
}

// Process performs actual stitching
func (i Inputs) Process() error {
	for _, p := range []string{i.Manifest, i.Config, i.Directory} {
		if !PathExists(p) {
			return fmt.Errorf("missing required argument")
		}
	}
	if len(i.OutName) == 0 {
		return fmt.Errorf("invalid output name information")
	}
	b, err := ioutil.ReadFile(i.Manifest)
	if err != nil {
		return err
	}
	m := &Manifest{}
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	if err := m.Check(); err != nil {
		return err
	}
	b, err = ioutil.ReadFile(i.Config)
	if err != nil {
		return err
	}
	cfg := &Exports{}
	if err := json.Unmarshal(b, &cfg); err != nil {
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
