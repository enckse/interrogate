package internal

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"mime"
	"net/http"
	"path/filepath"
)

const (
	// SessionKey contains the unique session identifier
	SessionKey = "session"
	// ClientKey contains the client connection information
	ClientKey = "client"
	// TimestampKey contains timestamp information
	TimestampKey = "timestamp"
	// ModeKey stores underlying save mode information
	ModeKey = "mode"
)

type (
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
	// PageData represents the templating for a survey page
	PageData struct {
		QueryParams string
		Title       string
		Session     string
		Snapshot    int
		Hidden      []Field
		Questions   []Field
	}
	// Configuration is the file-based configuration
	Configuration struct {
		Server struct {
			Questions string
			Bind      string
			Snapshot  int
			Storage   string
			Temp      string
			Resources string
			Tag       string
			Convert   bool
			Mask      struct {
				Admin   bool
				Enabled bool
			}
			Admin struct {
				User string
				Pass string
			}
		}
	}

	// ManifestEntry represents a line in the manifest
	ManifestEntry struct {
		Name   string
		Client string
		Mode   string
		Idx    int
	}

	// ManifestData is how we serialize the data to the manifest
	ManifestData struct {
		Title     string
		Tag       string
		File      string
		Manifest  []*ManifestEntry
		Warning   string
		ShowMasks bool
		Masks     []string
		Available []string
		CfgName   string
	}

	// Config represents the question configuration
	Config struct {
		Metadata  Meta       `yaml:"meta"`
		Questions []Question `yaml:"questions"`
	}

	// Meta represents a configuration overall survey meta-definition
	Meta struct {
		Title string `yaml:"title"`
	}

	// Question represents a single question configuration definition
	Question struct {
		Text        string   `yaml:"text"`
		Description string   `yaml:"desc"`
		Type        string   `yaml:"type"`
		Attributes  []string `yaml:"attrs"`
		Options     []string `yaml:"options"`
		Numbered    int      `yaml:"numbered"`
		Basis       string   `yaml:"basis"`
		Height      string   `yaml:"height"`
		Width       string   `yaml:"width"`
		Group       string   `yaml:"group"`
	}

	// ResultData is the resulting data from a submission
	ResultData struct {
		Datum map[string][]string `json:"data"`
	}

	// Exports are fields that are exported for reporting/display
	Exports struct {
		Fields []*ExportField `json:"fields"`
	}

	// Manifest represents the actual object-definition of the manifest
	Manifest struct {
		Files   []string `json:"files"`
		Clients []string `json:"clients"`
		Modes   []string `json:"modes"`
	}

	// ExportField is how fields are exported for definition
	ExportField struct {
		Text string `json:"text"`
		Type string `json:"type"`
	}
)

// Write writes the manifest to file
func (manifest *Manifest) Write(filename string) {
	datum, err := json.Marshal(manifest)
	if err != nil {
		Error("unable to marshal manifest", err)
		return
	}
	if err := ioutil.WriteFile(filename, datum, 0644); err != nil {
		Error("manifest writing failure", err)
	}
}

// NewManifest is responsible for creating a new manifest
func NewManifest(contents []byte) (*Manifest, error) {
	var manifest Manifest
	if err := json.Unmarshal(contents, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

// Check validates a manifest
func (manifest *Manifest) Check() error {
	valid := true
	if len(manifest.Files) != len(manifest.Clients) {
		valid = false
	}
	if len(manifest.Files) != len(manifest.Modes) {
		valid = false
	}
	if valid {
		return nil
	}
	return fmt.Errorf("corrupt index")
}

// NewPageData creates new page data objects
func NewPageData(req *http.Request, snapshot int) *PageData {
	pd := &PageData{}
	pd.QueryParams = req.URL.RawQuery
	pd.Snapshot = snapshot
	if len(pd.QueryParams) > 0 {
		pd.QueryParams = fmt.Sprintf("?%s", pd.QueryParams)
	}
	return pd
}

// SetHidden marks a field as hidden
func (f *Field) SetHidden() {
	f.hidden = true
}

// Hidden indicates if the field is hidden
func (f *Field) Hidden() bool {
	return f.hidden
}

// HandleTemplate executes a page data-based template
func (pd *PageData) HandleTemplate(resp http.ResponseWriter, tmpl *template.Template) {
	if err := tmpl.Execute(resp, pd); err != nil {
		Error("template execution error", err)
	}
}

// ReadManifestFile reads a manifest from file definitions
func ReadManifestFile(dir, tag string) (string, *Manifest, error) {
	existing := &Manifest{}
	fname := filepath.Join(dir, fmt.Sprintf("%s.index.manifest", tag))
	if PathExists(fname) {
		c, err := ioutil.ReadFile(fname)
		if err != nil {
			Error("unable to read index", err)
			return fname, nil, err
		}
		existing, err = NewManifest(c)
		if err != nil {
			Error("corrupt index", err)
			return fname, nil, err
		}
		if err := existing.Check(); err != nil {
			Info("invalid index... (lengths)")
			return fname, nil, fmt.Errorf("invalid index lengths")
		}
	}
	return fname, existing, nil
}

// IsAdmin checks if something is admin only
func IsAdmin(token string, req *http.Request) bool {
	query := req.URL.Query()
	v, ok := query["token"]
	if !ok {
		return false
	}
	if len(v) > 0 {
		for _, value := range v {
			if value == token {
				return true
			}
		}
	}
	return false
}

// GetStaticResource reads a static resource and configures response appropriately
func GetStaticResource(staticPath, staticURL string, resp http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	full := filepath.Join(staticPath, path)
	notFound := true
	var b []byte
	var err error
	m := mime.TypeByExtension(filepath.Ext(path))
	if m == "" {
		m = "text/plaintext"
	}
	resp.Header().Set("Content-Type", m)
	if PathExists(full) {
		b, err = ioutil.ReadFile(full)
		if err == nil {
			notFound = false
		} else {
			Error(fmt.Sprintf("%s asset read failure: %v", path), err)
		}
	}
	if notFound {
		b, err = ReadAssetRaw(filepath.Join(staticURL, path))
		if err != nil {
			resp.WriteHeader(http.StatusNotFound)
			return
		}
	}
	resp.Write(b)
}
