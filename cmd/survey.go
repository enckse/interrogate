package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/epiphyte/goutils/config"
	"github.com/epiphyte/goutils/logger"
	"github.com/epiphyte/goutils/opsys"
)

var (
	lock = &sync.Mutex{}
)

const (
	staticURL        = "/static/"
	surveyURL        = "/survey/"
	tokenParam       = "token"
	surveyClientURL  = surveyURL + "%d/%s"
	alphaNum         = "abcdefghijklmnopqrstuvwxyz0123456789"
	beginURL         = "/begin/"
	indexFile        = "index.manifest"
	questionFileName = "questions"
	qReset           = "RESET"
	saveFileName     = "save"
)

func readContent(directory string, name string) string {
	file := filepath.Join(directory, name)
	b, err := ioutil.ReadFile(file)
	if err != nil {
		logger.WriteError("unable to read file: "+file, err)
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
		logger.WriteError("unable to read template: "+file, err)
		panic("bad template")
	}
	return t
}

func handleTemplate(resp http.ResponseWriter, tmpl *template.Template, pd *PageData) {
	err := tmpl.Execute(resp, pd)
	if err != nil {
		logger.WriteError("template execution error", err)
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

func getTuple(req *http.Request, strPos int) (string, bool) {
	path := req.URL.Path
	parts := strings.Split(path, "/")
	required := strPos
	if len(parts) < required+1 {
		logger.WriteInfo("warning, invalid url", path)
		return "", false
	}
	return parts[strPos], true
}

func writeString(file *os.File, line string) {
	if _, err := file.WriteString(line); err != nil {
		logger.WriteError("file append error", err)
	}
}

func createPath(filename string, ctx *Context) string {
	return filepath.Join(ctx.store, filename)
}

func newFile(filename string, ctx *Context) (*os.File, error) {
	fname := createPath(filename, ctx)
	logger.WriteInfo("file name", fname)
	return os.OpenFile(fname, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
}

func responseBadRequest(resp http.ResponseWriter, message string, err error) {
	logger.WriteInfo("bad request")
	if err != nil {
		logger.WriteError("request error", err)
	}
	http.Error(resp, message, 400)
}

func readManifestFile(ctx *Context) (string, *Manifest, error) {
	existing := &Manifest{}
	fname := createPath(fmt.Sprintf("%s.%s", ctx.tag, indexFile), ctx)
	if opsys.PathExists(fname) {
		logger.WriteInfo("reading index")
		c, err := ioutil.ReadFile(fname)
		if err != nil {
			logger.WriteError("unable to read index", err)
			return fname, nil, err
		}
		existing, err = readManifest(c)
		if err != nil {
			logger.WriteError("corrupt index", err)
			return fname, nil, err
		}
		err = existing.Check()
		if err != nil {
			logger.WriteWarn("invalid index... (lengths)")
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
			curMode := existing.Modes[i]
			update := false
			// if we currently have a 'save' we only overwrite with another 'save'
			if curMode == saveFileName {
				if mode == saveFileName {
					update = true
				}
			} else {
				update = true
			}
			if update {
				existing.Files[i] = filename
				existing.Modes[i] = mode
			}
			handled = true
			break
		}
	}
	if !handled {
		existing.Clients = append(existing.Clients, client)
		existing.Files = append(existing.Files, filename)
		existing.Modes = append(existing.Modes, mode)
	}
	logger.WriteInfo("writing new index", fname)
	writeManifest(existing, fname)
}

func timeString() string {
	return time.Now().Format("2006-01-02T15-04-05")
}

func saveData(data map[string][]string, ctx *Context, mode string, client string, session string) {
	name := ""
	for _, c := range strings.ToLower(fmt.Sprintf("%s_%s_%s", client, getSession(6), session)) {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || (c == '_') {
			name = name + string(c)
		}
	}
	data["client"] = []string{client}
	ts := timeString()
	data["timestamp"] = []string{ts}
	fname := fmt.Sprintf("%s_%s_%s_%s", ctx.tag, ts, mode, name)
	go reindex(client, fname, ctx, mode)
	j, jerr := newFile(fname+JsonFile, ctx)
	if jerr == nil {
		defer j.Close()
		jsonString, merr := json.Marshal(data)
		if merr == nil {
			j.Write(jsonString)
		} else {
			logger.WriteError("unable to write json", merr)
		}
	} else {
		logger.WriteError("error writing json output", jerr)
	}
}

func getClient(req *http.Request) string {
	remoteAddress := req.RemoteAddr
	host, _, err := net.SplitHostPort(remoteAddress)
	if err == nil {
		remoteAddress = host
	} else {
		logger.WriteError("unable to read host port", err)
	}
	return remoteAddress
}

func saveEndpoint(resp http.ResponseWriter, req *http.Request, ctx *Context) {
	mode, valid := getTuple(req, 1)
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

	go saveData(datum, ctx, mode, getClient(req), sess)
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
	v, ok := query[tokenParam]
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
	restarting := false
	for k, v := range req.Form {
		switch k {
		case "questions":
			name := v[0]
			if name == qReset {
				name = ctx.available[0]
			}
			q := filepath.Join(ctx.temp, questionFileName)
			err := ioutil.WriteFile(q, []byte(name), 0644)
			if err != nil {
				logger.WriteError("unable to write question file and restart", err)
			}
		case "restart":
			for _, val := range v {
				if val == "on" {
					restarting = true
				}
			}
		}
	}
	if restarting {
		logger.WriteInfo("restart requested")
		os.Exit(1)
	}
	lock.Lock()
	defer lock.Unlock()
	pd := &ManifestData{}
	pd.Available = ctx.available
	pd.Available = append(pd.Available, qReset)
	pd.Token = ctx.token
	f, m, err := readManifestFile(ctx)
	pd.Title = "Admin"
	pd.Tag = ctx.tag
	pd.File = f
	pd.CfgName = ctx.cfgName
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
		logger.WriteError("template execution error", err)
	}
}

func dispResults(resp http.ResponseWriter, req *http.Request, ctx *Context) {
	if !isAdmin(ctx, req) {
		return
	}
	lock.Lock()
	defer lock.Unlock()
	_, m, werr := readManifestFile(ctx)
	if werr == nil {
		results := filepath.Join(ctx.temp, fmt.Sprintf("survey.%s", timeString()))
		err := stitch(m, ctx.store, results, true)
		if err == nil {
			htmlResult := results + htmlFile
			err = convFormat(results, ctx.cfgName, htmlResult)
			if err == nil {
				data, err := ioutil.ReadFile(htmlResult)
				if err == nil {
					resp.Write(data)
				} else {
					logger.WriteError("unable to read stitch results", err)
					werr = err
				}
			} else {
				logger.WriteError("unable to convert format", err)
				werr = err
			}
		} else {
			logger.WriteError("unable to stitch", err)
			werr = err
		}
	} else {
		logger.WriteError("unable to get manifest", werr)
	}
	if werr != nil {
		resp.Write([]byte("unable to process results"))
	}
}

func surveyEndpoint(resp http.ResponseWriter, req *http.Request, ctx *Context) {
	sess, valid := getTuple(req, 2)
	if !valid {
		return
	}
	pd := NewPageData(req, ctx)
	pd.Session = sess
	query := req.URL.Query()
	for _, q := range ctx.questions {
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
	pd.Title = ctx.title
	handleTemplate(resp, ctx.surveyTmpl, pd)
}

func main() {
	rand.Seed(time.Now().UnixNano())
	bind := flag.String("bind", "0.0.0.0:8080", "binding (ip:port)")
	tag := flag.String("tag", timeString(), "output tag")
	configFile := flag.String("config", "settings.conf", "configuration path")
	flag.Parse()
	cfg := *configFile
	logging := logger.NewLogOptions()
	logging.Info = true
	logger.ConfigureLogging(logging)
	logger.WriteInfo(vers)
	conf, err := config.LoadConfig(cfg, config.NewConfigSettings())
	if err != nil {
		logger.Fatal("unable to load config", err)
	}
	tmp := conf.GetStringOrDefault("temp", "/tmp/")
	questionFile := filepath.Join(tmp, questionFileName)
	existed := opsys.PathExists(questionFile)
	questions := ""
	if existed {
		logger.WriteInfo("loading question set input file", questionFile)
		q, err := ioutil.ReadFile(questionFile)
		if err != nil {
			logger.Fatal("unable to read question setting file", err)
		}
		questions = string(q)
	}
	if err != nil {
		if existed {
			logger.Fatal("unable to remove question file", err)
		} else {
			logger.WriteDebug("unable to remove non-existing file")
		}
	}
	initialQuestions := conf.GetStringOrEmpty("questions")
	if questions == "" {
		questions = initialQuestions
	}
	if strings.TrimSpace(questions) == "" {
		panic("no question set?")
	}
	dir := filepath.Dir(cfg)
	preOver := conf.GetStringOrEmpty("pre")
	ignore := make(map[string]struct{})
	if preOver != "" {
		ignore[preOver] = struct{}{}
		preOver = filepath.Join(dir, fmt.Sprintf("%s%s", preOver, questionConf))
	}
	postOver := conf.GetStringOrEmpty("post")
	if postOver != "" {
		ignore[postOver] = struct{}{}
		postOver = filepath.Join(dir, fmt.Sprintf("%s%s", postOver, questionConf))
	}
	ignore[initialQuestions] = struct{}{}
	settingsFile := filepath.Join(dir, questions)
	runSurvey(conf, &initSurvey{
		bind:        *bind,
		tag:         *tag,
		tmp:         tmp,
		questions:   settingsFile,
		inQuestions: initialQuestions,
		preOverlay:  preOver,
		postOverlay: postOver,
		searchDir:   dir,
		ignores:     ignore,
	})
}

type initSurvey struct {
	bind        string
	tag         string
	tmp         string
	inQuestions string
	questions   string
	preOverlay  string
	postOverlay string
	searchDir   string
	ignores     map[string]struct{}
}

func runSurvey(conf *config.Config, settings *initSurvey) {
	static := conf.GetStringOrDefault("resources", "/usr/share/survey/resources/")
	snapValue := conf.GetIntOrDefaultOnly("snapshot", 15)
	ctx := &Context{}
	ctx.snapshot = snapValue
	ctx.tag = conf.GetStringOrDefault("tag", settings.tag)
	ctx.store = conf.GetStringOrDefault("storage", defaultStore)
	ctx.temp = settings.tmp
	ctx.staticPath = staticURL
	ctx.beginTmpl = readTemplate(static, "begin.html")
	ctx.surveyTmpl = readTemplate(static, "survey.html")
	ctx.completeTmpl = readTemplate(static, "complete.html")
	ctx.adminTmpl = readTemplate(static, "admin.html")
	ctx.token = conf.GetStringOrDefault("token", time.Now().Format("150405"))
	ctx.available = []string{settings.inQuestions}
	ctx.cfgName = settings.questions
	avails, err := ioutil.ReadDir(settings.searchDir)
	if err != nil {
		logger.Fatal("unable to read available configs", err)
	}
	for _, a := range avails {
		base := filepath.Base(a.Name())
		if strings.HasSuffix(base, questionConf) {
			base = strings.Replace(base, questionConf, "", -1)
			if _, ok := settings.ignores[base]; !ok {
				ctx.available = append(ctx.available, base)
			}
		}
	}
	logger.WriteInfo("admin token", ctx.token)
	for _, d := range []string{ctx.store, ctx.temp} {
		err := os.MkdirAll(d, 0755)
		if err != nil {
			logger.Fatal("unable to create directory", err)
		}
	}
	ctx.load(settings.questions, settings.preOverlay, settings.postOverlay)
	http.HandleFunc("/", func(resp http.ResponseWriter, req *http.Request) {
		homeEndpoint(resp, req, ctx)
	})
	http.HandleFunc(surveyURL, func(resp http.ResponseWriter, req *http.Request) {
		surveyEndpoint(resp, req, ctx)
	})
	http.HandleFunc("/completed", func(resp http.ResponseWriter, req *http.Request) {
		completeEndpoint(resp, req, ctx)
	})
	http.HandleFunc("/results", func(resp http.ResponseWriter, req *http.Request) {
		dispResults(resp, req, ctx)
	})
	http.HandleFunc("/admin", func(resp http.ResponseWriter, req *http.Request) {
		adminEndpoint(resp, req, ctx)
	})
	for _, v := range []string{saveFileName, "snapshot"} {
		http.HandleFunc(fmt.Sprintf("/%s/", v), func(resp http.ResponseWriter, req *http.Request) {
			saveEndpoint(resp, req, ctx)
		})
	}
	staticPath := filepath.Join(static, staticURL)
	http.Handle(staticURL, http.StripPrefix(staticURL, http.FileServer(http.Dir(staticPath))))
	err = http.ListenAndServe(conf.GetStringOrDefault("bind", settings.bind), nil)
	if err != nil {
		logger.WriteError("unable to start", err)
		panic("failure")
	}
}
