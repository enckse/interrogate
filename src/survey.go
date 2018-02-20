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
)

const Version = "2.0.0"

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

const staticURL = "/static/"

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
	begin := readTemplate(*static, "begin.html")
	http.HandleFunc("/", func(resp http.ResponseWriter, req *http.Request) {
		err := begin.Execute(resp, NewPageData(req))
		if err != nil {
			log.Print("begin template error")
			log.Print(err)
		}
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
