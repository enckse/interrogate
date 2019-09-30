package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
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
	staticURL        = "/static/"
	surveyURL        = "/survey/"
	surveyClientURL  = surveyURL + "%d/%s"
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
		http.Handler
		snapshot     int
		tag          string
		store        string
		temp         string
		beginTmpl    *template.Template
		surveyTmpl   *template.Template
		completeTmpl *template.Template
		adminTmpl    *template.Template
		questions    []internal.Field
		title        string
		staticPath   string
		token        string
		available    []string
		cfgName      string
		memoryConfig string
		serveStatic  string
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
)

func (ctx *Context) newSet(configFile string) error {
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return err
	}
	var config internal.Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return err
	}
	ctx.title = config.Metadata.Title
	var mapping []internal.Field
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
		field := &internal.Field{}
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
			field.SetHidden()
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
				field.MinSize = internal.SetIfEmpty(field.Basis, fmt.Sprintf("%d", min))
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
			field.Basis = internal.SetIfEmpty(field.Basis, "50")
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
			field.Height = internal.SetIfEmpty(field.Height, "250")
			field.Width = internal.SetIfEmpty(field.Width, "250")
		}
		field.Group = q.Group
		field.RawType = internal.CreateHash(-1, q.Type)
		field.Hash = internal.CreateHash(field.ID, field.Text)
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
	exportConf := filepath.Join(ctx.store, fmt.Sprintf("run.config.%s", internal.TimeString()))
	if err := ioutil.WriteFile(exportConf, datum, 0644); err != nil {
		return err
	}
	fmt.Println(fmt.Sprintf("running config: %s", exportConf))
	ctx.memoryConfig = exportConf
	return nil
}

func homeEndpoint(resp http.ResponseWriter, req *http.Request, ctx *Context) {
	pd := ctx.newPage(req)
	pd.Session = internal.NewSession(20)
	pd.HandleTemplate(resp, ctx.beginTmpl)
}

func completeEndpoint(resp http.ResponseWriter, req *http.Request, ctx *Context) {
	pd := ctx.newPage(req)
	pd.HandleTemplate(resp, ctx.completeTmpl)
}

func (ctx *Context) getManifest() (string, *internal.Manifest, error) {
	return internal.ReadManifestFile(ctx.store, ctx.tag)
}

func reindex(client, filename string, ctx *Context, mode string) {
	lock.Lock()
	defer lock.Unlock()
	handled := false
	fname, existing, err := ctx.getManifest()
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

func saveData(data *internal.ResultData, ctx *Context, mode string, client string, session string) {
	name := ""
	for _, c := range strings.ToLower(fmt.Sprintf("%s_%s_%s", client, internal.NewSession(6), session)) {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || (c == '_') {
			name = name + string(c)
		}
	}
	data.Datum[internal.ClientKey] = []string{client}
	ts := internal.TimeString()
	data.Datum[internal.TimestampKey] = []string{ts}
	fname := fmt.Sprintf("%s_%s_%s_%s", ctx.tag, ts, mode, name)
	go reindex(client, fname, ctx, mode)
	j, jerr := internal.NewFile(ctx.store, fname+".json")
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

func saveEndpoint(resp http.ResponseWriter, req *http.Request, ctx *Context) {
	mode, valid := internal.GetURLTuple(req, 1)
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
	go saveData(r, ctx, mode, internal.GetClient(req), sess)
}

func adminEndpoint(resp http.ResponseWriter, req *http.Request, ctx *Context) {
	if !internal.IsAdmin(ctx.token, req) {
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
			if err := ioutil.WriteFile(q, []byte(name), 0644); err != nil {
				internal.Error("unable to write question file and restart", err)
			}
		case "restart":
			restarting = internal.IsChecked(v)
		case "bundling":
			bundling = internal.IsChecked(v)
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
	f, m, err := ctx.getManifest()
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
	if err := ctx.adminTmpl.Execute(resp, pd); err != nil {
		internal.Error("template execution error", err)
	}
}

func bundle(ctx *Context, readResult string) []byte {
	lock.Lock()
	defer lock.Unlock()
	f, _, err := ctx.getManifest()
	if err != nil {
		internal.Error("unable to read bundle manifest", err)
		return nil
	}
	results := filepath.Join(ctx.temp, fmt.Sprintf("survey.%s", internal.TimeString()))
	internal.Info(fmt.Sprintf("result file: %s", results))
	inputs := internal.Inputs{
		Manifest:  f,
		OutName:   results,
		Directory: ctx.store,
		Config:    ctx.memoryConfig,
	}
	if err := inputs.Process(); err != nil {
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
	if !internal.IsAdmin(ctx.token, req) {
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

func (ctx *Context) newPage(req *http.Request) *internal.PageData {
	return internal.NewPageData(req, ctx.snapshot)
}

func surveyEndpoint(resp http.ResponseWriter, req *http.Request, ctx *Context) {
	sess, valid := internal.GetURLTuple(req, 2)
	if !valid {
		return
	}
	pd := ctx.newPage(req)
	pd.Session = sess
	query := req.URL.Query()
	for _, q := range ctx.questions {
		obj := q
		value, ok := query[q.Text]
		if ok && len(value) == 1 {
			obj.Value = value[0]
		}
		if obj.Hidden() {
			pd.Hidden = append(pd.Hidden, obj)
		} else {
			pd.Questions = append(pd.Questions, obj)
		}
	}
	pd.Title = ctx.title
	pd.HandleTemplate(resp, ctx.surveyTmpl)
}

func main() {
	bind := flag.String("bind", "0.0.0.0:8080", "binding (ip:port)")
	tag := flag.String("tag", internal.TimeString(), "output tag")
	configFile := flag.String("config", "settings.conf", "configuration path")
	flag.Parse()
	cfg := *configFile
	internal.Info(vers)
	conf := &internal.Configuration{}
	cfgData, err := ioutil.ReadFile(cfg)
	if err != nil {
		internal.Fatal("unable to load config bytes", err)
	}
	if err := yaml.Unmarshal(cfgData, conf); err != nil {
		internal.Fatal("unable to load config", err)
	}
	tmp, cwd := internal.ResolvePath(conf.Server.Temp, "")
	questionFile := filepath.Join(tmp, questionFileName)
	questions := ""
	if internal.PathExists(questionFile) {
		internal.Info(fmt.Sprintf("loading question set input file: %s", questionFile))
		q, err := ioutil.ReadFile(questionFile)
		if err != nil {
			internal.Fatal("unable to read questoin settings file", err)
		}
		questions = string(q)
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
	pathed, c := internal.ResolvePath(path, s.cwd)
	s.cwd = c
	return pathed
}

func (ctx *Context) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	internal.GetStaticResource(ctx.serveStatic, staticURL, resp, req)
}

func runSurvey(conf *internal.Configuration, settings *initSurvey) {
	baseAsset := internal.ReadAsset("base")
	baseTemplate, err := template.New("base").Parse(string(baseAsset))
	if err != nil {
		internal.Fatal("unable to parse base template", err)
	}
	snapValue := conf.Server.Snapshot
	ctx := &Context{}
	ctx.snapshot = snapValue
	ctx.tag = internal.SetIfEmpty(conf.Server.Tag, settings.tag)
	ctx.store = settings.resolvePath(conf.Server.Storage)
	ctx.store = filepath.Join(ctx.store, ctx.tag)
	ctx.temp = settings.tmp
	ctx.staticPath = staticURL
	ctx.serveStatic = settings.resolvePath(conf.Server.Resources)
	ctx.beginTmpl = internal.ReadTemplate(baseTemplate, "begin")
	ctx.surveyTmpl = internal.ReadTemplate(baseTemplate, "survey")
	ctx.completeTmpl = internal.ReadTemplate(baseTemplate, "complete")
	ctx.adminTmpl = internal.ReadTemplate(baseTemplate, "admin")
	ctx.token = internal.SetIfEmpty(conf.Server.Token, time.Now().Format("150405"))
	ctx.available = []string{settings.inQuestions}
	ctx.cfgName = settings.questions
	if conf.Server.Convert {
		if err := internal.ConvertJSON(settings.searchDir); err != nil {
			internal.Fatal("unable to convert configuration file", err)
		}
	}
	avails, err := ioutil.ReadDir(settings.searchDir)
	if err != nil {
		internal.Fatal("unable to read available configs", err)
	}
	for _, a := range avails {
		base := filepath.Base(a.Name())
		if strings.HasSuffix(base, internal.ConfigExt) {
			base = strings.Replace(base, internal.ConfigExt, "", -1)
			if base != settings.inQuestions {
				ctx.available = append(ctx.available, base)
			}
		}
	}
	internal.Info(fmt.Sprintf("admin token: %s", ctx.token))
	for _, d := range []string{ctx.store, ctx.temp} {
		if err := os.MkdirAll(d, 0755); err != nil {
			internal.Fatal("unable to create directory", err)
		}
	}
	if err := ctx.newSet(fmt.Sprintf("%s%s", settings.questions, internal.ConfigExt)); err != nil {
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
	http.Handle(staticURL, http.StripPrefix(staticURL, ctx))
	if err := http.ListenAndServe(internal.SetIfEmpty(conf.Server.Bind, settings.bind), nil); err != nil {
		internal.Fatal("unable to start", err)
	}
}
