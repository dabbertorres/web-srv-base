package mail

import (
	"bytes"
    "errors"
	htmlTemplate "html/template"
	"io"
	"net/smtp"
	"regexp"
	"sync"
	textTemplate "text/template"
)

const (
	serverAddr = "mail:smtp"
)

// wrapper template for both text/template and html/template
type template struct {
	subject  string
	template interface {
		Execute(w io.Writer, data interface{}) error
	}
}

type headerData struct {
	From    string
	To      string
	Subject string
}

var (
    NoTemplateError = errors.New("no such template loaded")

	mutex     sync.RWMutex
	templates = make(map[string]template)

	serverFromAddr string
	headers        = textTemplate.Must(textTemplate.New("headers").Parse("From: {{ .From }}\nTo: {{ .To }}\nSubject: {{ .Subject }}\n\n"))

	newlineCorrecter = regexp.MustCompile("\\r?\\n")
)

func From(from string) {
	serverFromAddr = from
}

func LoadTemplatePlain(name, subject, text string) error {
	t, err := textTemplate.New(name).Parse(text)
	if err != nil {
		return err
	}

	mutex.Lock()
	templates[name] = template{
		subject:  subject,
		template: t,
	}
	mutex.Unlock()
	return nil
}

func LoadTemplateHTML(name, subject, text string) error {
	t, err := htmlTemplate.New(name).Parse(text)
	if err != nil {
		return err
	}

	mutex.Lock()
	templates[name] = template{
		subject:  subject,
		template: t,
	}
	mutex.Unlock()
	return nil
}

func Send(templateName, to string, data interface{}) (err error) {
	buf := bytes.NewBuffer(nil)

	mutex.RLock()
	defer mutex.RUnlock()
	t, ok := templates[templateName]
    if !ok {
        return NoTemplateError
    }

	err = headers.Execute(buf, &headerData{
		From:    serverFromAddr,
		To:      to,
		Subject: t.subject,
	})
	if err != nil {
		return
	}

	err = t.template.Execute(buf, data)
	if err != nil {
		return
	}

	out := newlineCorrecter.ReplaceAllLiteral(buf.Bytes(), []byte{'\r', '\n'})
	err = smtp.SendMail(serverAddr, nil, serverFromAddr, []string{to}, out)
	return
}
