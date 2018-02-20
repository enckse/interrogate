package main

import (
	"flag"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
    "fmt"
    "math/rand"
)

const Version = "2.0.0"
const staticURL = "/static/"
const surveyURL = "/survey/%d/%s"

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
	snapshot int
	tag      string
	store    string
	config   string
	lock     *sync.Mutex
    beginTmpl *template.Template
    surveyTmpl *template.Template
}

type PageData struct {
	QueryParams string
	Title       string
}

func NewPageData(req *http.Request) *PageData {
	pd := &PageData{}
	pd.QueryParams = req.URL.RawQuery
	// TODO: handle title
	pd.Title = "Survey"
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
    pd := NewPageData(req)
    handleTemplate(resp, ctx.beginTmpl, pd)
}

const alphaNum = "abcdefghijklmnopqrstuvwxyz0123456789"

func getSession() string {
    alphaNumeric := []rune(alphaNum)
    b := make([]rune, 20)
    runes := len(alphaNumeric)
    for i := range b {
        b[i] = alphaNumeric[rand.Intn(runes)]
    }
    return string(b)
}

func surveyEndpoint(resp http.ResponseWriter, req *http.Request, ctx *Context) {
    //pd := NewPageData(req)
    //handleTemplate(resp, ctx.surveyTmpl, pd)
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
    //ctx.surveyTmpl = readTemplate(*static, "survey.html")
	http.HandleFunc("/", func(resp http.ResponseWriter, req *http.Request) {
        homeEndpoint(resp, req, ctx)
	})
    http.HandleFunc("/begin", func(resp http.ResponseWriter, req *http.Request) {
        http.Redirect(resp, req, fmt.Sprintf(surveyURL, 0, getSession()), http.StatusSeeOther)
    })
	staticPath := filepath.Join(*static, staticURL)
	log.Print(staticPath)
	http.Handle(staticURL, http.StripPrefix(staticURL, http.FileServer(http.Dir(staticPath))))
	err := http.ListenAndServe(*bind, nil)
	if err != nil {
		log.Print("unable to start survey process")
		log.Print(err)
		panic("failure")
	}
}
