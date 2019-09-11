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
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	yaml "gopkg.in/yaml.v2"
)

const (
	questionConf     = ".json"
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

// Context for the running server
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
	memoryConfig string
	stitcher     string
}

type initSurvey struct {
	bind        string
	tag         string
	tmp         string
	inQuestions string
	questions   string
	searchDir   string
	cwd         string
}

// Configuration is the file-based configuration
type Configuration struct {
	Server struct {
		Questions string
		Bind      string
		Snapshot  int
		Storage   string
		Temp      string
		Resources string
		Stitcher  string
		Tag       string
		Token     string
	}
}

// ExportField is how fields are exported for definition
type ExportField struct {
	Text string `json:"text"`
	Type string `json:"type"`
}

// Field represents a question field
type Field struct {
	Value       string
	ID          int
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

// ManifestEntry represents a line in the manifest
type ManifestEntry struct {
	Name   string
	Client string
	Mode   string
	Idx    int
}

// ManifestData is how we serialize the data to the manifest
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

// PageData represents the templating for a survey page
type PageData struct {
	QueryParams string
	Title       string
	Session     string
	Snapshot    int
	Hidden      []Field
	Questions   []Field
}

// Config represents the question configuration
type Config struct {
	Metadata  Meta       `json:"meta"`
	Questions []Question `json:"questions"`
}

// Meta represents a configuration overall survey meta-definition
type Meta struct {
	Title string `json:"title"`
}

// Manifest represents the actual object-definition of the manifest
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

// Question represents a single question configuration definition
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
	Group       string   `json:"group"`
}

func writeError(message string, err error) {
	fmt.Println(fmt.Sprintf("%s (%v)", message, err))
}

func fatal(message string, err error) {
	writeError(message, err)
	panic("fatal error ^")
}

func pathExists(file string) bool {
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return false
	}
	return true
}

func info(message string) {
	fmt.Println(message)
}

func writeManifest(manifest *Manifest, filename string) {
	datum, err := json.Marshal(manifest)
	if err != nil {
		writeError("unable to marshal manifest", err)
		return
	}
	err = ioutil.WriteFile(filename, datum, 0644)
	if err != nil {
		writeError("manifest writing failure", err)
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

func (ctx *Context) newSet(configFile string) error {
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return err
	}
	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return err
	}
	ctx.title = config.Metadata.Title
	var mapping []Field
	number := 0
	inCond := false
	condCount := 0
	var exports []*ExportField
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
		exports = append(exports, &ExportField{Text: field.Text, Type: q.Type})
	}
	if inCond {
		panic("unclosed conditional")
	}
	ctx.questions = mapping
	datum, err := json.Marshal(exports)
	if err != nil {
		writeError("unable to write memory config", err)
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

func convFormat(manifest, out, dir, stitcherBin, configFile string) error {
	cmd := exec.Command(stitcherBin, "--manifest", manifest, "--out", out, "--dir", dir, "--config", configFile)
	return cmd.Run()
}

func readAsset(name string) string {
	asset, err := Asset(fmt.Sprintf("templates/%s.html", name))
	if err != nil {
		fatal(fmt.Sprintf("template not available %s", name), err)
	}
	return string(asset)
}

func readTemplate(base, tmpl string) *template.Template {
	file := readAsset(tmpl)
	def := strings.Replace(base, "{{CONTENT}}", file, -1)
	t, err := template.New("t").Parse(def)
	if err != nil {
		fatal(fmt.Sprintf("unable to read file %s", file), err)
	}
	return t
}

func handleTemplate(resp http.ResponseWriter, tmpl *template.Template, pd *PageData) {
	err := tmpl.Execute(resp, pd)
	if err != nil {
		writeError("template execution error", err)
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
		info(fmt.Sprintf("warning, invalid url %s", path))
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

func readManifestFile(ctx *Context) (string, *Manifest, error) {
	existing := &Manifest{}
	fname := createPath(fmt.Sprintf("%s.index.manifest", ctx.tag), ctx)
	if pathExists(fname) {
		c, err := ioutil.ReadFile(fname)
		if err != nil {
			writeError("unable to read index", err)
			return fname, nil, err
		}
		existing, err = readManifest(c)
		if err != nil {
			writeError("corrupt index", err)
			return fname, nil, err
		}
		err = existing.check()
		if err != nil {
			info("invalid index... (lengths)")
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
		info(fmt.Sprintf("save %s", fname))
	}
	if jerr == nil {
		defer j.Close()
		jsonString, merr := json.Marshal(data)
		if merr == nil {
			j.Write(jsonString)
		} else {
			writeError("unable to write json", merr)
		}
	} else {
		writeError("error writing json output", jerr)
	}
}

func getClient(req *http.Request) string {
	remoteAddress := req.RemoteAddr
	host, _, err := net.SplitHostPort(remoteAddress)
	if err == nil {
		remoteAddress = host
	} else {
		writeError("unable to read host port", err)
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
				writeError("unable to write question file and restart", err)
			}
		case "restart":
			restarting = isChecked(v)
		case "bundling":
			bundling = isChecked(v)
		}
	}
	if bundling {
		info("bundling")
		bundle(ctx, "")
	}
	if restarting {
		info("restart requested")
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
			entry.Idx = i
			pd.Manifest = append(pd.Manifest, entry)
		}
	} else {
		pd.Warning = err.Error()
	}
	err = ctx.adminTmpl.Execute(resp, pd)
	if err != nil {
		writeError("template execution error", err)
	}
}

func bundle(ctx *Context, readResult string) []byte {
	lock.Lock()
	defer lock.Unlock()
	f, _, err := readManifestFile(ctx)
	if err != nil {
		writeError("unable to read bundle manifest", err)
		return nil
	}
	results := filepath.Join(ctx.temp, fmt.Sprintf("survey.%s", timeString()))
	info(fmt.Sprintf("result file: %s", results))
	err = convFormat(f, results, ctx.store, ctx.stitcher, ctx.memoryConfig)
	if err != nil {
		writeError("unable to stitch bundle", err)
		return nil
	}
	if len(readResult) > 0 {
		data, err := ioutil.ReadFile(fmt.Sprintf("%s.%s", results, readResult))
		if err != nil {
			writeError("unable to read result file", err)
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
		resp.Write(data)
	} else {
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
	info(vers)
	conf := &Configuration{}
	cfgData, err := ioutil.ReadFile(cfg)
	if err != nil {
		fatal("unable to load config bytes", err)
	}
	err = yaml.Unmarshal(cfgData, conf)
	if err != nil {
		fatal("unable to load config", err)
	}
	tmp, cwd := resolvePath(conf.Server.Temp, "")
	questionFile := filepath.Join(tmp, questionFileName)
	existed := pathExists(questionFile)
	questions := ""
	if existed {
		info(fmt.Sprintf("loading question set input file: %s", questionFile))
		q, err := ioutil.ReadFile(questionFile)
		if err != nil {
			fatal("unable to read questoin settings file", err)
		}
		questions = string(q)
	}
	if err != nil {
		if existed {
			fatal("unable to remove question file", err)
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
			writeError("unable to determine working directory", err)
			return path, c
		}
		info(fmt.Sprintf("cwd is %s", c))
	}
	return filepath.Join(c, path), c
}

func setIfEmpty(setting, defaultValue string) string {
	if strings.TrimSpace(setting) == "" {
		return defaultValue
	}
	return setting
}

func runSurvey(conf *Configuration, settings *initSurvey) {
	static := settings.resolvePath(conf.Server.Resources)
	snapValue := conf.Server.Snapshot
	ctx := &Context{}
	ctx.snapshot = snapValue
	ctx.tag = setIfEmpty(conf.Server.Tag, settings.tag)
	ctx.store = settings.resolvePath(conf.Server.Storage)
	ctx.store = filepath.Join(ctx.store, ctx.tag)
	ctx.temp = settings.tmp
	ctx.staticPath = staticURL
	ctx.stitcher = conf.Server.Stitcher
	baseTemplate := readAsset("base")
	ctx.beginTmpl = readTemplate(baseTemplate, "begin")
	ctx.surveyTmpl = readTemplate(baseTemplate, "survey")
	ctx.completeTmpl = readTemplate(baseTemplate, "complete")
	ctx.adminTmpl = readTemplate(baseTemplate, "admin")
	ctx.token = setIfEmpty(conf.Server.Token, time.Now().Format("150405"))
	ctx.available = []string{settings.inQuestions}
	ctx.cfgName = settings.questions
	avails, err := ioutil.ReadDir(settings.searchDir)
	if err != nil {
		fatal("unable to read available configs", err)
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
	info(fmt.Sprintf("admin token: %s", ctx.token))
	for _, d := range []string{ctx.store, ctx.temp} {
		err := os.MkdirAll(d, 0755)
		if err != nil {
			fatal("unable to create directory", err)
		}
	}
	err = ctx.newSet(fmt.Sprintf("%s%s", settings.questions, questionConf))
	if err != nil {
		fatal("unable to load question set", err)
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
	staticPath := filepath.Join(static, staticURL)
	http.Handle(staticURL, http.StripPrefix(staticURL, http.FileServer(http.Dir(staticPath))))
	err = http.ListenAndServe(setIfEmpty(conf.Server.Bind, settings.bind), nil)
	if err != nil {
		fatal("unable to start", err)
	}
}
