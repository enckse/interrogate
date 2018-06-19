package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/epiphyte/goutils"
)

var (
	lock = &sync.Mutex{}
)

const staticURL = "/static/"
const surveyURL = "/survey/"
const surveyClientURL = surveyURL + "%d/%s"
const alphaNum = "abcdefghijklmnopqrstuvwxyz0123456789"
const beginURL = "/begin/"
const uploadURL = "/upload"
const indexFile = "index.manifest"

func readContent(directory string, name string) string {
	file := filepath.Join(directory, name)
	b, err := ioutil.ReadFile(file)
	if err != nil {
		goutils.WriteError("unable to read file: "+file, err)
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
		goutils.WriteError("unable to read template: "+file, err)
		panic("bad template")
	}
	return t
}

func handleTemplate(resp http.ResponseWriter, tmpl *template.Template, pd *PageData) {
	pd.Following = pd.Follow < pd.set
	err := tmpl.Execute(resp, pd)
	if err != nil {
		goutils.WriteError("template execution error", err)
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
		goutils.WriteInfo("warning, invalid url", path)
		return "", 0, false
	}
	idx, err := strconv.Atoi(parts[intPos])
	if err != nil {
		goutils.WriteInfo("warning, invalid int", path)
		return "", 0, false
	}
	return parts[strPos], idx, true
}

func writeString(file *os.File, line string, upload []string) []string {
	upload = append(upload, line)
	if _, err := file.WriteString(line); err != nil {
		goutils.WriteError("file append error", err)
	}
	return upload
}

func uploadRequest(addr string, datum io.Reader) bool {
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s%s", addr, uploadURL), datum)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		goutils.WriteError("upload error", err)
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		return true
	} else {
		body, _ := ioutil.ReadAll(resp.Body)
		goutils.WriteDebug(string(body))
		return false
	}
}

func doUpload(addr string, filename string, data []string, raw map[string][]string) {
	j, err := NewUpload(filename, data, raw)
	if err != nil {
		goutils.WriteError("unable to upload", err)
		return
	}
	jBytes := bytes.NewBuffer(j)
	defer jBytes.Reset()
	tries := 0
	for {
		if uploadRequest(addr, jBytes) {
			goutils.WriteInfo("uploaded...")
			break
		}
		if tries >= 3 {
			goutils.WriteInfo("giving up...")
			break
		}
		sleep := time.Duration(rand.Intn(5))
		time.Sleep(sleep * time.Second)
		tries += 1
	}
}

func createPath(filename string, ctx *Context) string {
	return filepath.Join(ctx.store, filename)
}

func newFile(filename string, ctx *Context) (*os.File, error) {
	fname := createPath(filename, ctx)
	goutils.WriteInfo("file name", fname)
	return os.OpenFile(fname, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
}

func responseBadRequest(resp http.ResponseWriter, message string, err error) {
	goutils.WriteInfo("bad request")
	if err != nil {
		goutils.WriteError("request error", err)
	}
	http.Error(resp, message, 400)
}

func readManifestFile(ctx *Context) (string, *Manifest, error) {
	existing := &Manifest{}
	fname := createPath(fmt.Sprintf("%s.%s", ctx.tag, indexFile), ctx)
	if goutils.PathExists(fname) {
		goutils.WriteInfo("reading index")
		c, err := ioutil.ReadFile(fname)
		if err != nil {
			goutils.WriteError("unable to read index", err)
			return fname, nil, err
		}
		existing, err = readManifest(c)
		if err != nil {
			goutils.WriteError("corrupt index", err)
			return fname, nil, err
		}
		err = existing.Check()
		if err != nil {
			goutils.WriteWarn("invalid index... (lengths)")
			return fname, nil, errors.New("invalid index lengths")
		}
	}
	return fname, existing, nil
}

func reindex(client, filename string, ctx *Context, mode string) {
	lock.Lock()
	defer lock.Unlock()
	handled := false
	fname, existing, err := readManifestFile(ctx)
	if err != nil {
		return
	}
	for i, c := range existing.Clients {
		if c == client {
			existing.Files[i] = filename
			existing.Modes[i] = mode
			handled = true
			break
		}
	}
	if !handled {
		existing.Clients = append(existing.Clients, client)
		existing.Files = append(existing.Files, filename)
		existing.Modes = append(existing.Modes, mode)
	}
	goutils.WriteInfo("writing new index", fname)
	writeManifest(existing, fname)
}

func uploadEndpoint(resp http.ResponseWriter, req *http.Request, ctx *Context) {
	if req.Body == nil {
		responseBadRequest(resp, "no request body", nil)
		return
	}
	upload, err := DecodeUpload(req.Body)
	if err != nil {
		responseBadRequest(resp, "invalid json", err)
		return
	}
	fileName := fmt.Sprintf("%s_%s_upload_%s", ctx.tag, getSession(6), upload.FileName)
	go reindex(getClient(req), fileName, ctx, "upload")
	j, jerr := newFile(fileName+JsonFile, ctx)
	if jerr == nil {
		defer j.Close()
		j.Write([]byte(upload.Raw))
	} else {
		goutils.WriteError("json uploaded error", jerr)
	}
	f, err := newFile(fileName+MarkdownFile, ctx)
	if err != nil {
		responseBadRequest(resp, "file io", err)
	}
	defer f.Close()
	for _, d := range upload.Data {
		if _, err := f.WriteString(d); err != nil {
			goutils.WriteError("file append error", err)
		}
	}
}

func timeString() string {
	return time.Now().Format("2006-01-02T15-04-05")
}

func saveData(data map[string][]string, ctx *Context, mode string, idx int, client string, session string) {
	name := ""
	for _, c := range strings.ToLower(fmt.Sprintf("%s_%s_%s", client, getSession(6), session)) {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || (c == '_') {
			name = name + string(c)
		}
	}
	data["client"] = []string{client}
	fname := fmt.Sprintf("%s_%s_%s_%s", ctx.tag, timeString(), mode, name)
	go reindex(client, fname, ctx, mode)
	j, jerr := newFile(fname+JsonFile, ctx)
	if jerr == nil {
		defer j.Close()
		jsonString, merr := json.Marshal(data)
		if merr == nil {
			j.Write(jsonString)
		} else {
			goutils.WriteError("unable to write json", merr)
		}
	} else {
		goutils.WriteError("result writing json output", jerr)
	}
	f, err := newFile(fname+MarkdownFile, ctx)
	if err != nil {
		goutils.WriteError("result error", err)
		return
	}
	defer f.Close()
	questionNum := 1
	metaNum := 1
	mapping := ctx.questionMaps[idx]
	var metaSet []string
	var uploadSet []string
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
				uploadSet = writeString(f, l, uploadSet)
			} else {
				metaSet = append(metaSet, l)
			}
		}
	}
	for _, l := range metaSet {
		uploadSet = writeString(f, l, uploadSet)
	}
	if ctx.uploading && len(uploadSet) > 0 {
		go doUpload(ctx.upload, fname, uploadSet, data)
	}
}

func getClient(req *http.Request) string {
	remoteAddress := req.RemoteAddr
	host, _, err := net.SplitHostPort(remoteAddress)
	if err == nil {
		remoteAddress = host
	} else {
		goutils.WriteError("unable to read host port", err)
	}
	return remoteAddress
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

	go saveData(datum, ctx, mode, idx, getClient(req), sess)
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

func isAdmin(ctx *Context, req *http.Request) bool {
	query := req.URL.Query()
	v, ok := query["token"]
	if !ok {
		return false
	}
	if len(v) > 0 {
		for _, value := range v {
			if value == ctx.token {
				return true
			}
		}
	}
	return false
}

func adminEndpoint(resp http.ResponseWriter, req *http.Request, ctx *Context) {
	if !isAdmin(ctx, req) {
		return
	}
	req.ParseForm()
	for k, v := range req.Form {
		if k == "restart" {
			for _, val := range v {
				if val == "on" {
					goutils.WriteInfo("restart requested")
					os.Exit(1)
				}
			}
		}
	}
	lock.Lock()
	defer lock.Unlock()
	pd := &ManifestData{}
	f, m, err := readManifestFile(ctx)
	pd.Title = "Admin"
	pd.Tag = ctx.tag
	pd.File = f
	if err == nil {
		for i, obj := range m.Files {
			entry := &ManifestEntry{}
			entry.Name = obj
			entry.Client = m.Clients[i]
			entry.Mode = m.Modes[i]
			pd.Manifest = append(pd.Manifest, entry)
		}
	} else {
		pd.Warning = err.Error()
	}
	err = ctx.adminTmpl.Execute(resp, pd)
	if err != nil {
		goutils.WriteError("template execution error", err)
	}
}

func resultsEndpoint(resp http.ResponseWriter, req *http.Request, ctx *Context) {
	if !isAdmin(ctx, req) {
		return
	}
	lock.Lock()
	defer lock.Unlock()
	pd := &ManifestData{}
	_, m, err := readManifestFile(ctx)
	if err == nil {
		results := filepath.Join(ctx.temp, fmt.Sprintf("survey.%s", timeString()))
		err = stitch(m, MarkdownFile, ctx.store, results)
		if err == nil {
			data, err := ioutil.ReadFile(results + htmlFile)
			if err == nil {
				pd.Rendered = template.HTML(data)
			} else {
				goutils.WriteError("unable to read stitch results", err)
				pd.Warning = err.Error()
			}
		} else {
			goutils.WriteError("unable to stitch", err)
			pd.Warning = err.Error()
		}
	} else {
		pd.Warning = err.Error()
	}
	err = ctx.resultsTmpl.Execute(resp, pd)
	if err != nil {
		goutils.WriteError("template execution error", err)
	}
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
	handleTemplate(resp, ctx.surveyTmpl, pd)
}

func main() {
	storagePath := StoragePath
	configFile := ConfigFile
	tmpl := Resources
	rand.Seed(time.Now().UnixNano())
	bind := flag.String("bind", "0.0.0.0:8080", "binding (ip:port)")
	snapshot := flag.Int("snapshot", 15, "auto snapshot (<= 0 is disabled)")
	tag := flag.String("tag", timeString(), "output tag")
	store := flag.String("store", storagePath, "storage path for results")
	temp := flag.String("temp", TempDir, "working/processing dir")
	config := flag.String("config", configFile, "configuration path")
	staticResources := flag.String("static", tmpl, "static resource location")
	upload := flag.String("upload", "", "upload address (ip:port)")
	var questions strFlagSlice
	flag.Var(&questions, "questions", "question set (multiple allowed)")
	flag.Parse()
	logging := goutils.NewLogOptions()
	logging.Info = true
	goutils.ConfigureLogging(logging)
	goutils.WriteInfo(vers)
	settingsFile := configFile + "settings.conf"
	conf := &goutils.Config{}
	if !goutils.PathNotExists(settingsFile) {
		c, err := goutils.LoadConfig(settingsFile, goutils.NewConfigSettings())
		if err != nil {
			goutils.WriteError("settings error", err)
			panic("unable to read settings file")
		}
		conf = c
		for _, q := range conf.GetArrayOrEmpty("questions") {
			questions = append(questions, q)
		}
	}
	overrides := filepath.Join(configFile, "resources")
	resourceLocation := *staticResources
	if goutils.PathExists(overrides) {
		goutils.WriteInfo("using override location", overrides)
		resourceLocation = overrides
	}
	static := conf.GetStringOrDefault("static", resourceLocation)
	useSnap := conf.GetStringOrEmpty("snapshot")
	snapValue := *snapshot
	if len(useSnap) > 0 {
		snap, err := strconv.Atoi(useSnap)
		if err != nil {
			goutils.WriteError("unable to use snapshot setting", err)
			panic("unable to read snapshot setting")
		}
		snapValue = snap
	}

	ctx := &Context{}
	ctx.snapshot = snapValue
	ctx.tag = conf.GetStringOrDefault("tag", *tag)
	ctx.store = conf.GetStringOrDefault("store", *store)
	ctx.temp = conf.GetStringOrDefault("temp", *temp)
	ctx.config = *config
	ctx.upload = *upload
	ctx.uploading = len(ctx.upload) > 0
	ctx.staticPath = staticURL
	ctx.beginTmpl = readTemplate(static, "begin.html")
	ctx.surveyTmpl = readTemplate(static, "survey.html")
	ctx.completeTmpl = readTemplate(static, "complete.html")
	ctx.adminTmpl = readTemplate(static, "admin.html")
	ctx.resultsTmpl = readTemplate(static, "results.html")
	ctx.token = time.Now().Format("150405")
	goutils.WriteInfo("admin token", ctx.token)
	err := os.MkdirAll(ctx.store, 0644)
	if err != nil {
		goutils.WriteError("unable to create storage dir", err)
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
	http.HandleFunc(uploadURL, func(resp http.ResponseWriter, req *http.Request) {
		uploadEndpoint(resp, req, ctx)
	})
	http.HandleFunc("/results", func(resp http.ResponseWriter, req *http.Request) {
		resultsEndpoint(resp, req, ctx)
	})
	http.HandleFunc("/admin", func(resp http.ResponseWriter, req *http.Request) {
		adminEndpoint(resp, req, ctx)
	})
	for _, v := range []string{"save", "snapshot"} {
		http.HandleFunc(fmt.Sprintf("/%s/", v), func(resp http.ResponseWriter, req *http.Request) {
			saveEndpoint(resp, req, ctx)
		})
	}
	staticPath := filepath.Join(static, staticURL)
	http.Handle(staticURL, http.StripPrefix(staticURL, http.FileServer(http.Dir(staticPath))))
	err = http.ListenAndServe(conf.GetStringOrDefault("bind", *bind), nil)
	if err != nil {
		goutils.WriteError("unable to start", err)
		panic("failure")
	}
}
