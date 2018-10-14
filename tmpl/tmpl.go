package tmpl

import (
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/dabbertorres/web-srv-base/view"
)

const (
	PagesDir     = "pages"
	TemplatesDir = "templates"
)

var (
	templates *template.Template
)

func Load(appPath string) (err error) {
	// non-nil empty template
	templates = template.New("base").Option("missingkey=zero")

	walk := func(path string, info os.FileInfo, err error) error {
		if err == nil && info.Mode().IsRegular() && info.Size() > 0 {
			// drop the appPath prefix and the file extension for use as the page name
			relPath, _ := filepath.Rel(appPath, path)
			relPath = strings.TrimSuffix(relPath, filepath.Ext(relPath))

			// if we got the index/home page, change it's name to just "/"
			if relPath == "/index" || relPath == "/home" {
				relPath = "/"
			}

			buf, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			_, err = templates.New(relPath).Parse(string(buf))
		}

		return err
	}

	err = filepath.Walk(filepath.Join(appPath, TemplatesDir), walk)
	if err != nil {
		return
	}

	err = filepath.Walk(filepath.Join(appPath, PagesDir), walk)
	if err != nil {
		return
	}

	return
}

func Build(page string, w io.Writer, data view.Data) error {
	return templates.ExecuteTemplate(w, page, data)
}

func Pages() <-chan string {
	ch := make(chan string)

	go func(ch chan<- string) {
		for _, t := range templates.Templates() {
			ch <- t.Name()
		}
		close(ch)
	}(ch)

	return ch
}
