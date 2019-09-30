package internal

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v2"
)

const (
	// ConfigExt is the configuration file extension for questions
	ConfigExt = ".yaml"
	alphaNum  = "abcdefghijklmnopqrstuvwxyz0123456789"
)

// PathExists checks if a file exists
func PathExists(file string) bool {
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return false
	}
	return true
}

// CreateHash creates an internal hash identifier for a field in the survey to be 'unique'-ish
func CreateHash(number int, value string) string {
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

// TimeString gets the current time as a YYYY-HH-MMTHH-MM-SS string
func TimeString() string {
	return time.Now().Format("2006-01-02T15-04-05")
}

// NewSession creates a new survey session (unique identifier)
func NewSession(length int) string {
	alphaNumeric := []rune(alphaNum)
	b := make([]rune, length)
	runes := len(alphaNumeric)
	for i := range b {
		b[i] = alphaNumeric[rand.Intn(runes)]
	}
	return string(b)
}

// IsChecked will see if input form values indicate a checkbox is 'on'
func IsChecked(values []string) bool {
	for _, val := range values {
		if val == "on" {
			return true
		}
	}
	return false
}

// ResolvePath will determine a the working directory to the best of its ability
func ResolvePath(path string, cwd string) (string, string) {
	if strings.HasPrefix(path, "/") {
		return path, cwd
	}
	c := cwd
	if c == "" {
		c, err := os.Getwd()
		if err != nil {
			Error("unable to determine working directory", err)
			return path, c
		}
		Info(fmt.Sprintf("cwd is %s", c))
	}
	return filepath.Join(c, path), c
}

// SetIfEmpty will return a default value if a given input is empty
func SetIfEmpty(setting, defaultValue string) string {
	if strings.TrimSpace(setting) == "" {
		return defaultValue
	}
	return setting
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

// ConvertJSON perform json -> yaml configuration conversions
func ConvertJSON(search string) error {
	conv, err := ioutil.ReadDir(search)
	if err != nil {
		return err
	}
	for _, f := range conv {
		n := f.Name()
		if strings.HasSuffix(n, ".json") {
			y := fmt.Sprintf("%s%s", strings.TrimSuffix(n, ".json"), ConfigExt)
			if PathExists(y) {
				continue
			}
			Info(fmt.Sprintf("converting: %s", n))
			b, err := ioutil.ReadFile(filepath.Join(search, n))
			if err != nil {
				return err
			}
			var obj interface{}
			if err := json.Unmarshal(b, &obj); err != nil {
				return err
			}
			obj = convertMap(obj)
			b, err = yaml.Marshal(obj)
			if err != nil {
				return err
			}
			if err := ioutil.WriteFile(filepath.Join(search, y), b, 0644); err != nil {
				return err
			}
			Info(fmt.Sprintf("converted: %s", y))
		}
	}
	return nil
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

// ReadAssetRaw reads an asset and returns the raw data
func ReadAssetRaw(name string) ([]byte, error) {
	fixed := name
	if !strings.HasPrefix(fixed, "/") {
		fixed = fmt.Sprintf("/%s", fixed)
	}
	return Asset(fmt.Sprintf("templates%s", fixed))
}

// ReadAsset reads an asset and fails if not able to recover
func ReadAsset(name string) string {
	asset, err := ReadAssetRaw(fmt.Sprintf("%s.html", name))
	if err != nil {
		Fatal(fmt.Sprintf("template not available %s", name), err)
	}
	return string(asset)
}

// ReadTemplate reads a template resource
func ReadTemplate(base *template.Template, tmpl string) *template.Template {
	copied, err := base.Clone()
	if err != nil {
		Fatal("unable to clone base template", err)
	}
	file := ReadAsset(tmpl)
	t, err := copied.Parse(string(file))
	if err != nil {
		Fatal(fmt.Sprintf("unable to read file %s", file), err)
	}
	return t
}

// NewFile prepares a file for (append?) write
func NewFile(dir, filename string) (*os.File, error) {
	fname := filepath.Join(dir, filename)
	return os.OpenFile(fname, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
}

// GetURLTuple extracts path information from a URL request
func GetURLTuple(req *http.Request, strPos int) (string, bool) {
	path := req.URL.Path
	parts := strings.Split(path, "/")
	required := strPos
	if len(parts) < required+1 {
		Info(fmt.Sprintf("warning, invalid url %s", path))
		return "", false
	}
	return parts[strPos], true
}

// GetClient gets a request client address
func GetClient(req *http.Request) string {
	remoteAddress := req.RemoteAddr
	host, _, err := net.SplitHostPort(remoteAddress)
	if err == nil {
		remoteAddress = host
	} else {
		Error("unable to read host port", err)
	}
	return remoteAddress
}
