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
	"sort"
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

func writeString(file *os.File, line string, upload []string) {
    upload = append(upload, line)
	if _, err := file.WriteString(line); err != nil {
		log.Print("file append error")
		log.Print(err)
	}
}

func doUpload(addr string, filename string, data []string) {
}

func uploadEndpoint(resp http.ResponseWriter, req *http.Request, ctx *Context) {
}

func saveData(data map[string][]string, ctx *Context, mode string, idx int, client string, session string) {
	name := ""
	for _, c := range strings.ToLower(fmt.Sprintf("%s_%s_%s", client, getSession(6), session)) {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || (c == '_') {
			name = name + string(c)
		}
	}
    fname := fmt.Sprintf("%s_%s_%s_%s", ctx.tag, time.Now().Format("2006-01-02T15-04-05"), mode, name)
	filename := filepath.Join(ctx.store, fname)
	log.Print(filename)
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		log.Print("result writing error")
		log.Print(err)
		return
	}
	defer f.Close()
	questionNum := 1
	metaNum := 1
	mapping := ctx.questionMaps[idx]
	var metaSet []string
    var uploadSet []string
	data["client"] = []string{client}
	var keys []string
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := data[k]
		useNumber := 0
		questionType := k
		qType, ok := mapping[k]
		if ok {
			useNumber = questionNum
			questionType = qType
			questionNum += 1
		} else {
			useNumber = metaNum
			questionType = fmt.Sprintf("system (%s)", k)
			metaNum += 1
		}

		var localLines []string
		localLines = append(localLines, fmt.Sprintf("#### %d. %s\n\n```\n", useNumber, questionType))
		noAnswer := true
		for _, value := range v {
			if len(strings.TrimSpace(value)) == 0 {
				continue
			}
			noAnswer = false
			localLines = append(localLines, fmt.Sprintf("%s\n", value))
		}
		if noAnswer {
			localLines = append(localLines, "<no response>\n")
		}
		localLines = append(localLines, fmt.Sprintf("```\n\n"))
		for _, l := range localLines {
			if ok {
				writeString(f, l, uploadSet)
			} else {
				metaSet = append(metaSet, l)
			}
		}
	}
	for _, l := range metaSet {
		writeString(f, l, uploadSet)
	}
    if ctx.uploading && len(uploadSet) > 0 {
        go doUpload(ctx.upload, fname, uploadSet)
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
	upload := flag.String("upload", "", "upload address (ip:port)")
	var questions strFlagSlice
	flag.Var(&questions, "questions", "question set (multiple allowed)")
	flag.Parse()
	ctx := &Context{}
	ctx.lock = &sync.Mutex{}
	ctx.snapshot = *snapshot
	ctx.tag = *tag
	ctx.store = *store
	ctx.config = *config
	ctx.upload = *upload
    ctx.uploading = len(ctx.upload) > 0
	ctx.beginTmpl = readTemplate(*static, "begin.html")
	ctx.surveyTmpl = readTemplate(*static, "survey.html")
	ctx.completeTmpl = readTemplate(*static, "complete.html")
	err := os.MkdirAll(ctx.store, 0644)
	if err != nil {
		log.Print("unable to create storage directory")
		log.Print(err)
		return
	}
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
    http.HandleFunc("/upload", func(resp http.ResponseWriter, req *http.Request) {
        uploadEndpoint(resp, req, ctx)
    })
	for _, v := range []string{"save", "snapshot"} {
		http.HandleFunc(fmt.Sprintf("/%s/", v), func(resp http.ResponseWriter, req *http.Request) {
			saveEndpoint(resp, req, ctx)
		})
	}
	staticPath := filepath.Join(*static, staticURL)
	http.Handle(staticURL, http.StripPrefix(staticURL, http.FileServer(http.Dir(staticPath))))
	err = http.ListenAndServe(*bind, nil)
	if err != nil {
		log.Print("unable to start survey process")
		log.Print(err)
		panic("failure")
	}
}
