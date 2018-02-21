package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
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
	lock         *sync.Mutex
	beginTmpl    *template.Template
	surveyTmpl   *template.Template
	completeTmpl *template.Template
	pages        int
	questions    [][]Field
	titles       []string
	anons        []bool
	questionMaps []map[string]string
	upload       string
	uploading    bool
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
}

type Config struct {
	Metadata  Meta       `json:"meta"`
	Questions []Question `json:"questions"`
}

type Meta struct {
	Title string `json:"title"`
	Anon  string `json:"anon"`
}

type Question struct {
	Text        string   `json:"text"`
	Description string   `json:"desc"`
	Type        string   `json:"type"`
	Attributes  []string `json:"attrs"`
	Options     []string `json:"options"`
	Numbered    int      `json:"numbered"`
}

func DecodeUpload(reader io.Reader) (*UploadData, error) {
	var uploaded UploadData
	err := json.NewDecoder(reader).Decode(&uploaded)
	return &uploaded, err
}

func NewUpload(filename string, data []string) ([]byte, error) {
	datum := &UploadData{FileName: filename, Data: data}
	return json.Marshal(datum)
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
		questionMap[strconv.Itoa(k)] = fmt.Sprintf("%s (%s)", q.Text, q.Type)
		field.Description = q.Description
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
		case "slide":
			field.Slider = true
			field.SlideId = template.JS(fmt.Sprintf("slide%d", k))
			field.SlideHideId = template.JS(fmt.Sprintf("shide%d", k))
		default:
			panic("unknown question type: " + q.Type)
		}
		mapping = append(mapping, *field)
	}
	ctx.questionMaps = append(ctx.questionMaps, questionMap)
	ctx.questions = append(ctx.questions, mapping)
	return nil
}

func (ctx *Context) load(questions strFlagSlice) {
	if len(questions) == 0 {
		log.Print("no question sets given")
		panic("no questions!")
	}

	pos := 0
	for _, q := range questions {
		conf := filepath.Join(ctx.config, q+".config")
		err := ctx.newSet(conf, pos)
		pos = pos + 1
		if err != nil {
			log.Print("unable to load question set")
			log.Print(conf)
			log.Print(err)
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
