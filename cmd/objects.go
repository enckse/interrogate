package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/epiphyte/goutils"
)

const (
	JsonFile     = ".json"
	MarkdownFile = ".md"
)

type strFlagSlice []string

func (s *strFlagSlice) Set(str string) error {
	*s = append(*s, str)
	return nil
}

func (s *strFlagSlice) String() string {
	return fmt.Sprintf("%v", *s)
}

type Context struct {
	snapshot     int
	tag          string
	store        string
	config       string
	beginTmpl    *template.Template
	surveyTmpl   *template.Template
	completeTmpl *template.Template
	adminTmpl    *template.Template
	pages        int
	questions    [][]Field
	titles       []string
	anons        []bool
	questionMaps []map[string]string
	upload       string
	uploading    bool
	staticPath   string
}

type Field struct {
	Value       string
	Id          int
	Text        string
	Input       bool
	Long        bool
	Label       bool
	Check       bool
	Number      bool
	Explanation bool
	Description string
	Option      bool
	Slider      bool
	Required    string
	Options     []string
	SlideId     template.JS
	SlideHideId template.JS
	hidden      bool
	Basis       string
	Image       bool
	Video       bool
	Audio       bool
	Height      string
	Width       string
}

type ManifestEntry struct {
	Name   string
	Client string
	Mode   string
}

type ManifestData struct {
	Title    string
	Tag      string
	File     string
	Manifest []*ManifestEntry
	Warning  string
}

type PageData struct {
	QueryParams string
	Title       string
	Index       int
	Following   bool
	Follow      int
	Session     string
	Snapshot    int
	Anonymous   bool
	Hidden      []Field
	Questions   []Field
	set         int
}

type UploadData struct {
	FileName string
	Data     []string
	Raw      string
}

type Config struct {
	Metadata  Meta       `json:"meta"`
	Questions []Question `json:"questions"`
}

type Meta struct {
	Title string `json:"title"`
	Anon  string `json:"anon"`
}

type Manifest struct {
	Files   []string `json:"files"`
	Clients []string `json:"clients"`
	Modes   []string `json:"modes"`
}

func (m *Manifest) Check() error {
	valid := true
	if len(m.Files) != len(m.Clients) {
		valid = false
	}
	if len(m.Files) != len(m.Modes) {
		valid = false
	}
	if valid {
		return nil
	}
	return errors.New("corrupt index")
}

type Question struct {
	Text        string   `json:"text"`
	Description string   `json:"desc"`
	Type        string   `json:"type"`
	Attributes  []string `json:"attrs"`
	Options     []string `json:"options"`
	Numbered    int      `json:"numbered"`
	Basis       string   `json:"basis"`
	Height      string   `json:"height"`
	Width       string   `json:"width"`
}

func DecodeUpload(reader io.Reader) (*UploadData, error) {
	var uploaded UploadData
	err := json.NewDecoder(reader).Decode(&uploaded)
	return &uploaded, err
}

func NewUpload(filename string, data []string, raw map[string][]string) ([]byte, error) {
	rawString, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	datum := &UploadData{FileName: filename, Data: data, Raw: string(rawString)}
	return json.Marshal(datum)
}

func writeManifest(manifest *Manifest, filename string) {
	datum, err := json.Marshal(manifest)
	if err != nil {
		goutils.WriteError("unable to marshal manifest", err)
		return
	}
	err = ioutil.WriteFile(filename, datum, 0644)
	if err != nil {
		goutils.WriteError("manifest writing failure", err)
	}
}

func readManifest(contents []byte) (*Manifest, error) {
	var manifest Manifest
	err := json.Unmarshal(contents, &manifest)
	if err != nil {
		return nil, err
	}
	return &manifest, nil
}

func (ctx *Context) newSet(configFile string, position int) error {
	jfile, err := os.Open(configFile)
	if err != nil {
		return err
	}

	defer jfile.Close()
	data, err := ioutil.ReadAll(jfile)
	if err != nil {
		return err
	}
	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return err
	}
	ctx.titles = append(ctx.titles, config.Metadata.Title)
	ctx.anons = append(ctx.anons, config.Metadata.Anon != "FALSE")
	var mapping []Field
	questionMap := make(map[string]string)
	number := 0
	for _, q := range config.Questions {
		k := number
		number = number + 1
		if q.Numbered > 0 {
			k = q.Numbered
		}
		field := &Field{}
		for _, attr := range q.Attributes {
			if attr == "required" {
				field.Required = attr
			}
		}
		field.Id = k
		field.Text = q.Text
		field.Basis = q.Basis
		field.Height = q.Height
		field.Width = q.Width
		questionMap[strconv.Itoa(k)] = fmt.Sprintf("%s (%s)", q.Text, q.Type)
		field.Description = q.Description
		defaultDimensions := false
		switch q.Type {
		case "input":
			field.Input = true
		case "hidden":
			field.hidden = true
		case "long":
			field.Long = true
		case "option":
			field.Option = true
			field.Options = q.Options
		case "label":
			field.Label = true
		case "checkbox":
			field.Check = true
		case "number":
			field.Number = true
		case "image":
			field.Image = true
			defaultDimensions = true
		case "audio":
			field.Audio = true
		case "video":
			defaultDimensions = true
			field.Video = true
		case "slide":
			field.Slider = true
			field.SlideId = template.JS(fmt.Sprintf("slide%d", k))
			field.SlideHideId = template.JS(fmt.Sprintf("shide%d", k))
			field.Basis = getWhenEmpty(field.Basis, "50")
		default:
			panic("unknown question type: " + q.Type)
		}
		if field.Image || field.Audio || field.Video {
			field.Basis = fmt.Sprintf("%s%s", ctx.staticPath, field.Basis)
		}
		if defaultDimensions {
			field.Height = getWhenEmpty(field.Height, "250")
			field.Width = getWhenEmpty(field.Width, "250")
		}
		mapping = append(mapping, *field)
	}
	ctx.questionMaps = append(ctx.questionMaps, questionMap)
	ctx.questions = append(ctx.questions, mapping)
	return nil
}

func getWhenEmpty(value, dflt string) string {
	if len(strings.TrimSpace(value)) == 0 {
		return dflt
	} else {
		return value
	}
}

func (ctx *Context) load(questions strFlagSlice) {
	if len(questions) == 0 {
		goutils.WriteInfo("no questions given!")
		panic("no questions!")
	}

	pos := 0
	for _, q := range questions {
		conf := filepath.Join(ctx.config, q+".config")
		err := ctx.newSet(conf, pos)
		pos = pos + 1
		goutils.WriteDebug("config", conf)
		if err != nil {
			goutils.WriteError("unable to load question set", err)
			panic("invalid question set")
		}
	}
	ctx.pages = pos
}

func NewPageData(req *http.Request, ctx *Context) *PageData {
	pd := &PageData{}
	pd.QueryParams = req.URL.RawQuery
	pd.Snapshot = ctx.snapshot
	pd.set = ctx.pages
	if len(pd.QueryParams) > 0 {
		pd.QueryParams = fmt.Sprintf("?%s", pd.QueryParams)
	}
	return pd
}

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
	err = ioutil.WriteFile(out, b.Bytes(), 0644)
	if err != nil {
		return err
	}
	if isMarkdown {
		markdown := fmt.Sprintf("python -m markdown -x markdown.extensions.nl2br -x markdown.extensions.fenced_code -x markdown.extensions.tables %s > %s.html", out, out)
		_, err = goutils.RunBashCommand(markdown)
		return err
	}
	return nil
}
