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

const (
	questionConf     = ".config"
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

type Context struct {
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
	preManifest  string
	postManifest string
	memoryConfig string
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

type ExportField struct {
	Text string `json:"text"`
	Type string `json:"type"`
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
}

type ManifestEntry struct {
	Name   string
	Client string
	Mode   string
}

type ManifestData struct {
	Title     string
	Tag       string
	File      string
	Manifest  []*ManifestEntry
	Warning   string
	Available []string
	Token     string
	CfgName   string
}

type PageData struct {
	QueryParams string
	Title       string
	Session     string
	Snapshot    int
	Hidden      []Field
	Questions   []Field
}

type Config struct {
	Metadata  Meta       `json:"meta"`
	Questions []Question `json:"questions"`
}

type Meta struct {
	Title string `json:"title"`
}

type Manifest struct {
	Files   []string `json:"files"`
	Clients []string `json:"clients"`
	Modes   []string `json:"modes"`
}

func (m *Manifest) check() error {
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

func writeManifest(manifest *Manifest, filename string) {
	datum, err := json.Marshal(manifest)
	if err != nil {
		logger.WriteError("unable to marshal manifest", err)
		return
	}
	err = ioutil.WriteFile(filename, datum, 0644)
	if err != nil {
		logger.WriteError("manifest writing failure", err)
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

func (ctx *Context) newSet(configFile, pre, post string) error {
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return err
	}
	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return err
	}
	for idx, over := range []string{pre, post} {
		if over == "" {
			continue
		}
		if opsys.PathNotExists(over) {
			panic("overlay file not found: " + over)
		}
		var c Config
		oData, err := ioutil.ReadFile(over)
		if err != nil {
			return err
		}
		err = json.Unmarshal(oData, &c)
		if err != nil {
			return err
		}
		appending := c.Questions
		switch idx {
		case 0:
			appending = config.Questions
			config.Questions = c.Questions
			ctx.preManifest = over
		case 1:
			// this is valid but no-op
			ctx.postManifest = over
		default:
			panic("invalid overlay setting")
		}
		for _, oQuestion := range appending {
			config.Questions = append(config.Questions, oQuestion)
		}
	}
	ctx.title = config.Metadata.Title
	var mapping []Field
	number := 0
	inCond := false
	condCount := 0
	var exports []*ExportField
	for _, q := range config.Questions {
		condCount += 1
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
		case "hr":
			field.HorizontalFeed = true
		case "slide":
			field.Slider = true
			field.SlideId = template.JS(fmt.Sprintf("slide%d", k))
			field.SlideHideId = template.JS(fmt.Sprintf("shide%d", k))
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
		field.RawType = createHash(-1, q.Type)
		field.Hash = createHash(field.Id, field.Text)
		mapping = append(mapping, *field)
		exports = append(exports, &ExportField{Text: field.Text, Type: q.Type})
	}
	if inCond {
		panic("unclosed conditional")
	}
	ctx.questions = mapping
	datum, err := json.Marshal(exports)
	if err != nil {
		logger.WriteError("unable to write memory config", err)
		return err
	}
	exportConf := filepath.Join(ctx.store, fmt.Sprintf("run.config.%s", timeString()))
	err = ioutil.WriteFile(exportConf, datum, 0644)
	logger.WriteInfo("running config", exportConf)
	ctx.memoryConfig = exportConf
	return nil
}

func getWhenEmpty(value, dflt string) string {
	if len(strings.TrimSpace(value)) == 0 {
		return dflt
	} else {
		return value
	}
}

func NewPageData(req *http.Request, ctx *Context) *PageData {
	pd := &PageData{}
	pd.QueryParams = req.URL.RawQuery
	pd.Snapshot = ctx.snapshot
	if len(pd.QueryParams) > 0 {
		pd.QueryParams = fmt.Sprintf("?%s", pd.QueryParams)
	}
	return pd
}

func convFormat(manifest, out, dir, configFile string) error {
	_, err := opsys.RunBashCommand(fmt.Sprintf("survey-stitcher --manifest %s --out %s --dir %s --config %s", manifest, out, dir, configFile))
	return err
}

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

func createPath(filename string, ctx *Context) string {
	return filepath.Join(ctx.store, filename)
}

func newFile(filename string, ctx *Context) (*os.File, error) {
	fname := createPath(filename, ctx)
	logger.WriteDebug("file name", fname)
	return os.OpenFile(fname, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
}

func readManifestFile(ctx *Context) (string, *Manifest, error) {
	existing := &Manifest{}
	fname := createPath(fmt.Sprintf("%s.index.manifest", ctx.tag), ctx)
	if opsys.PathExists(fname) {
		logger.WriteDebug("reading index")
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
		err = existing.check()
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
	logger.WriteDebug("writing new index", fname)
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
	j, jerr := newFile(fname+".json", ctx)
	if mode == saveFileName {
		logger.WriteInfo("save", fname)
	}
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
	f, _, werr := readManifestFile(ctx)
	if werr == nil {
		results := filepath.Join(ctx.temp, fmt.Sprintf("survey.%s", timeString()))
		err := convFormat(f, results, ctx.store, ctx.memoryConfig)
		if err == nil {
			data, err := ioutil.ReadFile(results + ".html")
			if err == nil {
				resp.Write(data)
			} else {
				logger.WriteError("unable to read stitch results", err)
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

func runSurvey(conf *config.Config, settings *initSurvey) {
	static := conf.GetStringOrDefault("resources", "/usr/share/survey/resources/")
	snapValue := conf.GetIntOrDefaultOnly("snapshot", 15)
	ctx := &Context{}
	ctx.snapshot = snapValue
	ctx.tag = conf.GetStringOrDefault("tag", settings.tag)
	ctx.store = conf.GetStringOrDefault("storage", "/var/cache/survey/")
	ctx.store = filepath.Join(ctx.store, ctx.tag)
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
	logger.WriteDebug("questions", settings.questions)
	err = ctx.newSet(fmt.Sprintf("%s%s", settings.questions, questionConf), settings.preOverlay, settings.postOverlay)
	if err != nil {
		logger.WriteError("unable to load question set", err)
		panic("invalid question set")
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
