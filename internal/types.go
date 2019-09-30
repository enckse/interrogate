package internal

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
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
			Token     string
			Convert   bool
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
		Available []string
		Token     string
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