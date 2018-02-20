package main

import (
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
}

// NOTE this method is for translation only
func fakeData(pd *PageData) {
	pd.Title = "Survey"
	pd.Hidden = []Field{}
	pd.Questions = []Field{}
	pd.Hidden = append(pd.Hidden, Field{})
	pd.Hidden = append(pd.Hidden, Field{})
	pd.Questions = append(pd.Questions, Field{Input: true})
	pd.Questions = append(pd.Questions, Field{Label: true})
	pd.Questions = append(pd.Questions, Field{Long: true})
	pd.Questions = append(pd.Questions, Field{Explanation: true})
	pd.Questions = append(pd.Questions, Field{Check: true})
	pd.Questions = append(pd.Questions, Field{Number: true})
	pd.Questions = append(pd.Questions, Field{Option: true, Options: []string{"TEST"}})
	pd.Questions = append(pd.Questions, Field{Slider: true, SlideId: template.JS("slide1"), SlideHideId: template.JS("shide1"), Id: 1})
}

func NewPageData(req *http.Request, ctx *Context) *PageData {
	pd := &PageData{}
	pd.QueryParams = req.URL.RawQuery
	if len(pd.QueryParams) > 0 {
		pd.QueryParams = fmt.Sprintf("?%s", pd.QueryParams)
	}
	fakeData(pd)
	return pd
}

func handleTemplate(resp http.ResponseWriter, tmpl *template.Template, pd *PageData) {
	err := tmpl.Execute(resp, pd)
	if err != nil {
		log.Print("error executing template")
		log.Print(err)
	}
}

func homeEndpoint(resp http.ResponseWriter, req *http.Request, ctx *Context) {
	pd := NewPageData(req, ctx)
	pd.Session = getSession(20)
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
	sess, idx, valid := getTuple(req, 3, 2)
	if !valid {
		return
	}
	pd := NewPageData(req, ctx)
	pd.Session = sess
	pd.Index = idx
	pd.Follow = idx + 1
	if req.Method == "POST" {
		req.ParseForm()
		for k, _ := range req.Form {
			log.Print(k)
		}
	} else {
		handleTemplate(resp, ctx.surveyTmpl, pd)
	}
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
	http.HandleFunc("/", func(resp http.ResponseWriter, req *http.Request) {
		homeEndpoint(resp, req, ctx)
	})
	http.HandleFunc(surveyURL, func(resp http.ResponseWriter, req *http.Request) {
		surveyEndpoint(resp, req, ctx)
	})
	http.HandleFunc("/completed", func(resp http.ResponseWriter, req *http.Request) {
		completeEndpoint(resp, req, ctx)
	})
	http.HandleFunc("/snapshot/", func(resp http.ResponseWriter, req *http.Request) {
		saveEndpoint(resp, req, ctx)
	})
	staticPath := filepath.Join(*static, staticURL)
	http.Handle(staticURL, http.StripPrefix(staticURL, http.FileServer(http.Dir(staticPath))))
	err := http.ListenAndServe(*bind, nil)
	if err != nil {
		log.Print("unable to start survey process")
		log.Print(err)
		panic("failure")
	}
}
