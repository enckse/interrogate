package core

import (
	"fmt"
	"os"
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

// ResultData is the resulting data from a submission
type ResultData struct {
	Datum map[string][]string `json:"data"`
}

// Exports are fields that are exported for reporting/display
type Exports struct {
	Fields []*ExportField `json:"fields"`
}

// Manifest represents the actual object-definition of the manifest
type Manifest struct {
	Files   []string `json:"files"`
	Clients []string `json:"clients"`
	Modes   []string `json:"modes"`
}

// ExportField is how fields are exported for definition
type ExportField struct {
	Text string `json:"text"`
	Type string `json:"type"`
}

// PathExists checks if a file exists
func PathExists(file string) bool {
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return false
	}
	return true
}

// Check validates a manifest
func (m *Manifest) Check() error {
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
	return fmt.Errorf("corrupt index")
}
