package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const Version = "2.0.0"
const staticURL = "/static/"
const surveyURL = "/survey/"
const surveyClientURL = surveyURL + "%d/%s"
const alphaNum = "abcdefghijklmnopqrstuvwxyz0123456789"
const beginURL = "/begin/"

func readContent(directory string, name string) string {
	file := filepath.Join(directory, name)
	b, err := ioutil.ReadFile(file)
	if err != nil {
		log.Print("unable to read file: " + file)
		log.Print(err)
		panic("bad file")
	}
	return string(b)
}

func readTemplate(directory string, tmpl string) *template.Template {
	base := readContent(directory, "base.html")
	file := readContent(directory, tmpl)
	def := strings.Replace(base, "{{CONTENT}}", file, -1)
	t, err := template.New("t").Parse(def)
	if err != nil {
		log.Print("Unable to read template: " + file)
		log.Print(err)
		panic("bad template")
	}
	return t
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
	ctx.questions = append(ctx.questions, mapping)
	return nil
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

// NOTE this method is for translation only
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

func handleTemplate(resp http.ResponseWriter, tmpl *template.Template, pd *PageData) {
	pd.Following = pd.Follow < pd.set
	err := tmpl.Execute(resp, pd)
	if err != nil {
		log.Print("error executing template")
		log.Print(err)
	}
}

func homeEndpoint(resp http.ResponseWriter, req *http.Request, ctx *Context) {
	pd := NewPageData(req, ctx)
	pd.Session = getSession(20)
	pd.Index = 0
	handleTemplate(resp, ctx.beginTmpl, pd)
}

func completeEndpoint(resp http.ResponseWriter, req *http.Request, ctx *Context) {
	pd := NewPageData(req, ctx)
	handleTemplate(resp, ctx.completeTmpl, pd)
}

func getTuple(req *http.Request, strPos int, intPos int) (string, int, bool) {
	path := req.URL.Path
	parts := strings.Split(path, "/")
	required := strPos
	if intPos > strPos {
		required = intPos
	}
	if len(parts) < required+1 {
		log.Print("warning, invalid url")
		log.Print(path)
		return "", 0, false
	}
	idx, err := strconv.Atoi(parts[intPos])
	if err != nil {
		log.Print("invalid int value")
		log.Print(path)
		return "", 0, false
	}
	return parts[strPos], idx, true
}

func writeString(file *os.File, line string) {
	if _, err := file.WriteString(line); err != nil {
		log.Print("file append error")
		log.Print(err)
	}
}

func saveData(data map[string][]string, ctx *Context, mode string, idx int, client string, session string) {
	// TODO: directory needs to exist before we ever see this path
	name := ""
	for _, c := range strings.ToLower(fmt.Sprintf("%s_%s_%s", client, getSession(6), session)) {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || (c == '_') {
			name = name + string(c)
		}
	}
	filename := filepath.Join(ctx.store, fmt.Sprintf("%s_%s_%s_%s", ctx.tag, time.Now().Format("2006-01-02T15-04-05"), mode, name))
	log.Print(filename)
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		log.Print("result writing error")
		log.Print(err)
		return
	}
	defer f.Close()
	// TODO: need to map questions back from input results
	// TODO: Translate checkboxes
	for k, v := range data {
		writeString(f, fmt.Sprintf("#### %s\n\n```", k))
		for _, value := range v {
			if len(strings.TrimSpace(value)) == 0 {
				continue
			}
			writeString(f, fmt.Sprintf("%s\n", value))
		}
		writeString(f, fmt.Sprintf("```\n\n"))
	}
}

func saveEndpoint(resp http.ResponseWriter, req *http.Request, ctx *Context) {
	mode, idx, valid := getTuple(req, 1, 2)
	if !valid {
		return
	}
	req.ParseForm()
	datum := make(map[string][]string)
	sess := ""
	for k, v := range req.Form {
		datum[k] = v
		if k == "session" && len(v) > 0 {
			sess = v[0]
		}
	}

	go saveData(datum, ctx, mode, idx, req.RemoteAddr, sess)
}

func getSession(length int) string {
	alphaNumeric := []rune(alphaNum)
	b := make([]rune, length)
	runes := len(alphaNumeric)
	for i := range b {
		b[i] = alphaNumeric[rand.Intn(runes)]
	}
	return string(b)
}

func surveyEndpoint(resp http.ResponseWriter, req *http.Request, ctx *Context) {
	// TODO: map query parameters to question set/default
	sess, idx, valid := getTuple(req, 3, 2)
	if !valid {
		return
	}
	pd := NewPageData(req, ctx)
	pd.Session = sess
	pd.Index = idx
	pd.Follow = idx + 1
	if idx >= 0 && idx < len(ctx.questions) {
		questions := ctx.questions[idx]
		query := req.URL.Query()
		for _, q := range questions {
			obj := q
			value, ok := query[q.Text]
			if ok && len(value) == 1 {
				obj.Value = value[0]
			}
			if obj.hidden {
				pd.Hidden = append(pd.Hidden, obj)
			} else {
				pd.Questions = append(pd.Questions, obj)
			}
		}
		pd.Title = ctx.titles[idx]
		pd.Anonymous = ctx.anons[idx]
	}
	if req.Method == "POST" {
		req.ParseForm()
		for k, _ := range req.Form {
			log.Print(k)
		}
	} else {
		handleTemplate(resp, ctx.surveyTmpl, pd)
	}
}

type strFlagSlice []string

func (s *strFlagSlice) Set(str string) error {
	*s = append(*s, str)
	return nil
}

func (s *strFlagSlice) String() string {
	return fmt.Sprintf("%v", *s)
}

func main() {
	storagePath := "/var/cache/survey/"
	configFile := "/etc/survey/"
	tmpl := "/usr/share/survey/static/"
	if runtime.GOOS == "windows" {
		basePath := "C:\\survey\\"
		storagePath = basePath + "results\\"
		configFile = basePath + "config\\"
		tmpl = basePath + "static\\"
	}
	rand.Seed(time.Now().UnixNano())
	bind := flag.String("bind", "0.0.0.0:8080", "binding (ip:port)")
	snapshot := flag.Int("snapshot", 15, "auto snapshot (<= 0 is disabled)")
	tag := flag.String("tag", time.Now().Format("2006-01-02"), "output tag")
	store := flag.String("store", storagePath, "storage path for results")
	config := flag.String("config", configFile, "configuration path")
	static := flag.String("static", tmpl, "static resource location")
	var questions strFlagSlice
	flag.Var(&questions, "questions", "question set (multiple allowed)")
	flag.Parse()
	ctx := &Context{}
	ctx.lock = &sync.Mutex{}
	ctx.snapshot = *snapshot
	ctx.tag = *tag
	ctx.store = *store
	ctx.config = *config
	ctx.beginTmpl = readTemplate(*static, "begin.html")
	ctx.surveyTmpl = readTemplate(*static, "survey.html")
	ctx.completeTmpl = readTemplate(*static, "complete.html")
	ctx.load(questions)
	http.HandleFunc("/", func(resp http.ResponseWriter, req *http.Request) {
		homeEndpoint(resp, req, ctx)
	})
	http.HandleFunc(surveyURL, func(resp http.ResponseWriter, req *http.Request) {
		surveyEndpoint(resp, req, ctx)
	})
	http.HandleFunc("/completed", func(resp http.ResponseWriter, req *http.Request) {
		completeEndpoint(resp, req, ctx)
	})
	for _, v := range []string{"save", "snapshot"} {
		http.HandleFunc(fmt.Sprintf("/%s/", v), func(resp http.ResponseWriter, req *http.Request) {
			saveEndpoint(resp, req, ctx)
		})
	}
	staticPath := filepath.Join(*static, staticURL)
	http.Handle(staticURL, http.StripPrefix(staticURL, http.FileServer(http.Dir(staticPath))))
	err := http.ListenAndServe(*bind, nil)
	if err != nil {
		log.Print("unable to start survey process")
		log.Print(err)
		panic("failure")
	}
}
