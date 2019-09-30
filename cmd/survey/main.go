package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"math/rand"
	"mime"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	yaml "gopkg.in/yaml.v2"
	"voidedtech.com/survey/internal"
)

const (
	questionConf     = ".yaml"
	staticURL        = "/static/"
	surveyURL        = "/survey/"
	surveyClientURL  = surveyURL + "%d/%s"
	alphaNum         = "abcdefghijklmnopqrstuvwxyz0123456789"
	questionFileName = "questions"
	qReset           = "RESET"
	saveFileName     = "save"
)

var (
	lock = &sync.Mutex{}
	vers = "master"
)

type (
	// Context represent operating context
	Context struct {
		snapshot     int
		tag          string
		store        string
		temp         string
		beginTmpl    *template.Template
		surveyTmpl   *template.Template
		completeTmpl *template.Template
		adminTmpl    *template.Template
		questions    []Field
		title        string
		staticPath   string
		token        string
		available    []string
		cfgName      string
		memoryConfig string
	}

	// PageData represents the templating for a survey page
	PageData struct {
		QueryParams string
		Title       string
		Session     string
		Snapshot    int
		Hidden      []Field
		Questions   []Field
	}

	initSurvey struct {
		bind        string
		tag         string
		tmp         string
		inQuestions string
		questions   string
		searchDir   string
		cwd         string
	}

	// Field represents a question field
	Field struct {
		Value       string
		ID          int
		Text        string
		Input       bool
		Long        bool
		Label       bool
		Check       bool
		Number      bool
		Order       bool
		Explanation bool
		Description string
		Option      bool
		Slider      bool
		Required    string
		Options     []string
		Multi       bool
		MinSize     string
		SlideID     template.JS
		SlideHideID template.JS
		Basis       string
		Image       bool
		Video       bool
		Audio       bool
		Height      string
		Width       string
		// Control types, not input types
		CondStart      bool
		CondEnd        bool
		HorizontalFeed bool
		hidden         bool
		RawType        string
		Hash           string
		Group          string
	}

	staticHandler struct {
		http.Handler
		path string
	}
)

func createHash(number int, value string) string {
	use := "hash" + value
	if number >= 0 {
		use = fmt.Sprintf("%s%d", use, number)
	}
	use = strings.ToLower(use)
	output := ""
	for _, c := range use {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			output = output + string(c)
		}
	}
	return output
}

func (ctx *Context) newSet(configFile string) error {
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return err
	}
	var config internal.Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return err
	}
	ctx.title = config.Metadata.Title
	var mapping []Field
	number := 0
	inCond := false
	condCount := 0
	exports := &internal.Exports{}
	for _, q := range config.Questions {
		condCount++
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
		field.ID = k
		field.Text = q.Text
		field.Basis = q.Basis
		field.Height = q.Height
		field.Width = q.Width
		field.Description = q.Description
		defaultDimensions := false
		switch q.Type {
		case "input":
			field.Input = true
		case "hidden":
			field.hidden = true
		case "long":
			field.Long = true
		case "option", "multiselect":
			field.Option = true
			field.Options = q.Options
			field.Multi = q.Type == "multiselect"
			// NOTE: try a reasonable size of pixels
			if field.Multi {
				min := len(q.Options) * 20
				if min < 50 {
					min = 50
				}
				field.MinSize = getWhenEmpty(field.Basis, fmt.Sprintf("%d", min))
			}
		case "order":
			field.Order = true
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
		case "hr":
			field.HorizontalFeed = true
		case "slide":
			field.Slider = true
			field.SlideID = template.JS(fmt.Sprintf("slide%d", k))
			field.SlideHideID = template.JS(fmt.Sprintf("shide%d", k))
			field.Basis = getWhenEmpty(field.Basis, "50")
		case "conditional":
			if inCond {
				if condCount == 1 {
					panic("conditional contains no questions")
				}
				field.CondEnd = true
				inCond = false
			} else {
				condCount = 0
				inCond = true
				field.CondStart = true
			}
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
		field.Group = q.Group
		field.RawType = createHash(-1, q.Type)
		field.Hash = createHash(field.ID, field.Text)
		mapping = append(mapping, *field)
		exports.Fields = append(exports.Fields, &internal.ExportField{Text: field.Text, Type: q.Type})
	}
	if inCond {
		panic("unclosed conditional")
	}
	ctx.questions = mapping
	datum, err := json.Marshal(exports)
	if err != nil {
		internal.Error("unable to write memory config", err)
		return err
	}
	exportConf := filepath.Join(ctx.store, fmt.Sprintf("run.config.%s", timeString()))
	err = ioutil.WriteFile(exportConf, datum, 0644)
	fmt.Println(fmt.Sprintf("running config: %s", exportConf))
	ctx.memoryConfig = exportConf
	return nil
}

func getWhenEmpty(value, dflt string) string {
	if len(strings.TrimSpace(value)) == 0 {
		return dflt
	}
	return value
}

// NewPageData create a new survey page data object for templating
func NewPageData(req *http.Request, ctx *Context) *PageData {
	pd := &PageData{}
	pd.QueryParams = req.URL.RawQuery
	pd.Snapshot = ctx.snapshot
	if len(pd.QueryParams) > 0 {
		pd.QueryParams = fmt.Sprintf("?%s", pd.QueryParams)
	}
	return pd
}

func readAssetRaw(name string) ([]byte, error) {
	fixed := name
	if !strings.HasPrefix(fixed, "/") {
		fixed = fmt.Sprintf("/%s", fixed)
	}
	return internal.Asset(fmt.Sprintf("templates%s", fixed))
}

func readAsset(name string) string {
	asset, err := readAssetRaw(fmt.Sprintf("%s.html", name))
	if err != nil {
		internal.Fatal(fmt.Sprintf("template not available %s", name), err)
	}
	return string(asset)
}

func readTemplate(base *template.Template, tmpl string) *template.Template {
	copied, err := base.Clone()
	if err != nil {
		internal.Fatal("unable to clone base template", err)
	}
	file := readAsset(tmpl)
	t, err := copied.Parse(string(file))
	if err != nil {
		internal.Fatal(fmt.Sprintf("unable to read file %s", file), err)
	}
	return t
}

func handleTemplate(resp http.ResponseWriter, tmpl *template.Template, pd *PageData) {
	err := tmpl.Execute(resp, pd)
	if err != nil {
		internal.Error("template execution error", err)
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
		internal.Info(fmt.Sprintf("warning, invalid url %s", path))
		return "", false
	}
	return parts[strPos], true
}

func createPath(filename string, ctx *Context) string {
	return filepath.Join(ctx.store, filename)
}

func newFile(filename string, ctx *Context) (*os.File, error) {
	fname := createPath(filename, ctx)
	return os.OpenFile(fname, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
}

func readManifestFile(ctx *Context) (string, *internal.Manifest, error) {
	existing := &internal.Manifest{}
	fname := createPath(fmt.Sprintf("%s.index.manifest", ctx.tag), ctx)
	if internal.PathExists(fname) {
		c, err := ioutil.ReadFile(fname)
		if err != nil {
			internal.Error("unable to read index", err)
			return fname, nil, err
		}
		existing, err = internal.NewManifest(c)
		if err != nil {
			internal.Error("corrupt index", err)
			return fname, nil, err
		}
		err = existing.Check()
		if err != nil {
			internal.Info("invalid index... (lengths)")
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
	existing.Write(fname)
}

func timeString() string {
	return time.Now().Format("2006-01-02T15-04-05")
}

func saveData(data *internal.ResultData, ctx *Context, mode string, client string, session string) {
	name := ""
	for _, c := range strings.ToLower(fmt.Sprintf("%s_%s_%s", client, getSession(6), session)) {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || (c == '_') {
			name = name + string(c)
		}
	}
	data.Datum[internal.ClientKey] = []string{client}
	ts := timeString()
	data.Datum[internal.TimestampKey] = []string{ts}
	fname := fmt.Sprintf("%s_%s_%s_%s", ctx.tag, ts, mode, name)
	go reindex(client, fname, ctx, mode)
	j, jerr := newFile(fname+".json", ctx)
	if mode == saveFileName {
		internal.Info(fmt.Sprintf("save %s", fname))
	}
	if jerr == nil {
		defer j.Close()
		jsonString, merr := json.Marshal(data)
		if merr == nil {
			j.Write(jsonString)
		} else {
			internal.Error("unable to write json", merr)
		}
	} else {
		internal.Error("error writing json output", jerr)
	}
}

func getClient(req *http.Request) string {
	remoteAddress := req.RemoteAddr
	host, _, err := net.SplitHostPort(remoteAddress)
	if err == nil {
		remoteAddress = host
	} else {
		internal.Error("unable to read host port", err)
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
		if k == internal.SessionKey && len(v) > 0 {
			sess = v[0]
		}
	}

	r := &internal.ResultData{
		Datum: datum,
	}
	go saveData(r, ctx, mode, getClient(req), sess)
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

func isChecked(values []string) bool {
	for _, val := range values {
		if val == "on" {
			return true
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
	bundling := true
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
				internal.Error("unable to write question file and restart", err)
			}
		case "restart":
			restarting = isChecked(v)
		case "bundling":
			bundling = isChecked(v)
		}
	}
	if restarting {
		if bundling {
			internal.Info("bundling")
			bundle(ctx, "")
		}
		internal.Info("restart requested")
		os.Exit(1)
	}
	lock.Lock()
	defer lock.Unlock()
	pd := &internal.ManifestData{}
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
			entry := &internal.ManifestEntry{}
			entry.Name = obj
			entry.Client = m.Clients[i]
			entry.Mode = m.Modes[i]
			entry.Idx = i
			pd.Manifest = append(pd.Manifest, entry)
		}
	} else {
		pd.Warning = err.Error()
	}
	err = ctx.adminTmpl.Execute(resp, pd)
	if err != nil {
		internal.Error("template execution error", err)
	}
}

func bundle(ctx *Context, readResult string) []byte {
	lock.Lock()
	defer lock.Unlock()
	f, _, err := readManifestFile(ctx)
	if err != nil {
		internal.Error("unable to read bundle manifest", err)
		return nil
	}
	results := filepath.Join(ctx.temp, fmt.Sprintf("survey.%s", timeString()))
	internal.Info(fmt.Sprintf("result file: %s", results))
	inputs := internal.Inputs{
		Manifest:  f,
		OutName:   results,
		Directory: ctx.store,
		Config:    ctx.memoryConfig,
	}
	err = inputs.Process()
	if err != nil {
		internal.Error("unable to process results", err)
		return nil
	}
	if len(readResult) > 0 {
		data, err := ioutil.ReadFile(fmt.Sprintf("%s.%s", results, readResult))
		if err != nil {
			internal.Error("unable to read result file", err)
			return nil
		}
		return data
	}
	return nil
}

func getResults(resp http.ResponseWriter, req *http.Request, ctx *Context, display bool) {
	if !isAdmin(ctx, req) {
		return
	}
	fileResult := "tar.gz"
	if display {
		fileResult = "html"
	}
	data := bundle(ctx, fileResult)
	if data == nil {
		resp.Write([]byte("unable to process results"))
	} else {
		resp.Write(data)
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
	internal.Info(vers)
	conf := &internal.Configuration{}
	cfgData, err := ioutil.ReadFile(cfg)
	if err != nil {
		internal.Fatal("unable to load config bytes", err)
	}
	err = yaml.Unmarshal(cfgData, conf)
	if err != nil {
		internal.Fatal("unable to load config", err)
	}
	tmp, cwd := resolvePath(conf.Server.Temp, "")
	questionFile := filepath.Join(tmp, questionFileName)
	existed := internal.PathExists(questionFile)
	questions := ""
	if existed {
		internal.Info(fmt.Sprintf("loading question set input file: %s", questionFile))
		q, err := ioutil.ReadFile(questionFile)
		if err != nil {
			internal.Fatal("unable to read questoin settings file", err)
		}
		questions = string(q)
	}
	if err != nil {
		if existed {
			internal.Fatal("unable to remove question file", err)
		}
	}
	initialQuestions := conf.Server.Questions
	if questions == "" {
		questions = initialQuestions
	}
	if strings.TrimSpace(questions) == "" {
		panic("no question set?")
	}
	dir := filepath.Dir(cfg)
	settingsFile := filepath.Join(dir, questions)
	runSurvey(conf, &initSurvey{
		bind:        *bind,
		tag:         *tag,
		tmp:         tmp,
		questions:   settingsFile,
		inQuestions: initialQuestions,
		searchDir:   dir,
		cwd:         cwd,
	})
}

func (s *initSurvey) resolvePath(path string) string {
	pathed, c := resolvePath(path, s.cwd)
	s.cwd = c
	return pathed
}

func resolvePath(path string, cwd string) (string, string) {
	if strings.HasPrefix(path, "/") {
		return path, cwd
	}
	c := cwd
	if c == "" {
		c, err := os.Getwd()
		if err != nil {
			internal.Error("unable to determine working directory", err)
			return path, c
		}
		internal.Info(fmt.Sprintf("cwd is %s", c))
	}
	return filepath.Join(c, path), c
}

func setIfEmpty(setting, defaultValue string) string {
	if strings.TrimSpace(setting) == "" {
		return defaultValue
	}
	return setting
}

func (s *staticHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	full := filepath.Join(s.path, path)
	notFound := true
	var b []byte
	var err error
	m := mime.TypeByExtension(filepath.Ext(path))
	if m == "" {
		m = "text/plaintext"
	}
	resp.Header().Set("Content-Type", m)
	if internal.PathExists(full) {
		b, err = ioutil.ReadFile(full)
		if err == nil {
			notFound = false
		} else {
			internal.Error(fmt.Sprintf("%s asset read failure: %v", path), err)
		}
	}
	if notFound {
		b, err = readAssetRaw(filepath.Join(staticURL, path))
		if err != nil {
			resp.WriteHeader(http.StatusNotFound)
			return
		}
	}
	resp.Write(b)
}

func convertMap(i interface{}) interface{} {
	switch x := i.(type) {
	case map[interface{}]interface{}:
		m2 := map[string]interface{}{}
		for k, v := range x {
			m2[k.(string)] = convertMap(v)
		}
		return m2
	case []interface{}:
		for i, v := range x {
			x[i] = convertMap(v)
		}
	}
	return i
}

func convertJSON(search string) error {
	conv, err := ioutil.ReadDir(search)
	if err != nil {
		return err
	}
	for _, f := range conv {
		n := f.Name()
		if strings.HasSuffix(n, ".json") {
			y := fmt.Sprintf("%s%s", strings.TrimSuffix(n, ".json"), questionConf)
			if internal.PathExists(y) {
				continue
			}
			internal.Info(fmt.Sprintf("converting: %s", n))
			b, err := ioutil.ReadFile(filepath.Join(search, n))
			if err != nil {
				return err
			}
			var obj interface{}
			err = json.Unmarshal(b, &obj)
			if err != nil {
				return err
			}
			obj = convertMap(obj)
			b, err = yaml.Marshal(obj)
			if err != nil {
				return err
			}
			err = ioutil.WriteFile(filepath.Join(search, y), b, 0644)
			if err != nil {
				return err
			}
			internal.Info(fmt.Sprintf("converted: %s", y))
		}
	}
	return nil
}

func runSurvey(conf *internal.Configuration, settings *initSurvey) {
	baseAsset := readAsset("base")
	baseTemplate, err := template.New("base").Parse(string(baseAsset))
	if err != nil {
		internal.Fatal("unable to parse base template", err)
	}
	static := settings.resolvePath(conf.Server.Resources)
	snapValue := conf.Server.Snapshot
	ctx := &Context{}
	ctx.snapshot = snapValue
	ctx.tag = setIfEmpty(conf.Server.Tag, settings.tag)
	ctx.store = settings.resolvePath(conf.Server.Storage)
	ctx.store = filepath.Join(ctx.store, ctx.tag)
	ctx.temp = settings.tmp
	ctx.staticPath = staticURL
	ctx.beginTmpl = readTemplate(baseTemplate, "begin")
	ctx.surveyTmpl = readTemplate(baseTemplate, "survey")
	ctx.completeTmpl = readTemplate(baseTemplate, "complete")
	ctx.adminTmpl = readTemplate(baseTemplate, "admin")
	ctx.token = setIfEmpty(conf.Server.Token, time.Now().Format("150405"))
	ctx.available = []string{settings.inQuestions}
	ctx.cfgName = settings.questions
	if conf.Server.Convert {
		err = convertJSON(settings.searchDir)
		if err != nil {
			internal.Fatal("unable to convert configuration file", err)
		}
	}
	avails, err := ioutil.ReadDir(settings.searchDir)
	if err != nil {
		internal.Fatal("unable to read available configs", err)
	}
	for _, a := range avails {
		base := filepath.Base(a.Name())
		if strings.HasSuffix(base, questionConf) {
			base = strings.Replace(base, questionConf, "", -1)
			if base != settings.inQuestions {
				ctx.available = append(ctx.available, base)
			}
		}
	}
	internal.Info(fmt.Sprintf("admin token: %s", ctx.token))
	for _, d := range []string{ctx.store, ctx.temp} {
		err := os.MkdirAll(d, 0755)
		if err != nil {
			internal.Fatal("unable to create directory", err)
		}
	}
	err = ctx.newSet(fmt.Sprintf("%s%s", settings.questions, questionConf))
	if err != nil {
		internal.Fatal("unable to load question set", err)
	}
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
		getResults(resp, req, ctx, true)
	})
	http.HandleFunc("/bundle.tar.gz", func(resp http.ResponseWriter, req *http.Request) {
		getResults(resp, req, ctx, false)
	})
	http.HandleFunc("/admin", func(resp http.ResponseWriter, req *http.Request) {
		adminEndpoint(resp, req, ctx)
	})
	for _, v := range []string{saveFileName, "snapshot"} {
		http.HandleFunc(fmt.Sprintf("/%s/", v), func(resp http.ResponseWriter, req *http.Request) {
			saveEndpoint(resp, req, ctx)
		})
	}
	staticHandle := &staticHandler{path: static}
	http.Handle(staticURL, http.StripPrefix(staticURL, staticHandle))
	err = http.ListenAndServe(setIfEmpty(conf.Server.Bind, settings.bind), nil)
	if err != nil {
		internal.Fatal("unable to start", err)
	}
}
