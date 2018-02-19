package main

import (
    "net/http"
    "html/template"
    "path/filepath"
    "log"
)

const startTemplate = "begin.html"

func readTemplate(directory string, tmpl string) *template.Template {
    file := filepath.Join(directory, tmpl)
    t, err := template.ParseFiles(file)
    if err != nil {
        log.Print("Unable to read template: " + file)
        log.Print(err)
        panic("bad template")
    }
    return t
}

func main() {
    http.HandleFunc("/", func(resp http.ResponseWriter, req *http.Request) {
    })
}
