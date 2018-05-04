package tmpl

import (
	"html/template"
	"io"
	"os"
	"path/filepath"
	"io/ioutil"
	"strings"
)

var (
	templates *template.Template
)

func Load(templatesPath, pagesPath string) (err error) {
	// non-nil empty template
	templates = template.New("base").Option("missingkey=zero")

	walk := func(path string, info os.FileInfo, err error) error {
		if err == nil && info.Mode().IsRegular() && info.Size() > 0 {
			// drop the pagesPath prefix and the file extension for use as the page name
			relPath, _ := filepath.Rel(templatesPath, path)
			relPath = strings.TrimSuffix(relPath, filepath.Ext(relPath))

			// if we got the index page, change it's name to just "/"
			if relPath == "/index" {
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

	err = filepath.Walk(templatesPath, walk)
	if err != nil {
		return
	}

	err = filepath.Walk(pagesPath, walk)
	if err != nil {
		return
	}

	return
}

func Build(page string, w io.Writer, data interface{}) error {
	return templates.ExecuteTemplate(w, filepath.Base(page), data)
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
