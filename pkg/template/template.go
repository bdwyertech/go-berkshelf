package template

import (
	"bytes"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"text/template"

	log "github.com/sirupsen/logrus"

	"github.com/go-sprout/sprout"
	"github.com/go-sprout/sprout/group/all"
	"github.com/go-sprout/sprout/registry/crypto"
	"github.com/goccy/go-yaml"
	"github.com/google/uuid"
)

func New() *template.Template {
	handler := sprout.New()
	handler.AddGroups(all.RegistryGroup())
	handler.AddRegistry(crypto.NewRegistry())
	tfs := handler.Build() // template.FuncMap
	tfs["sha512sum"] = func(input string) string {
		hash := sha512.Sum512([]byte(input))
		return hex.EncodeToString(hash[:])
	}
	tfs["toYaml"] = func(i interface{}) string {
		buf := new(bytes.Buffer)
		enc := yaml.NewEncoder(buf)
		if err := enc.Encode(i); err != nil {
			log.Fatal(err)
		}
		return buf.String()
	}

	tfs["uuidv7"] = func() string {
		return uuid.Must(uuid.NewV7()).String()
	}

	tfs["fromFile"] = func(f string) string {
		fBytes, err := os.ReadFile(f)
		if err != nil {
			log.Fatal(err)
		}
		return string(fBytes)
	}

	tpl := template.New("")
	tfs["tpl"] = tplFun(tpl)
	tpl.Funcs(tfs)

	return tpl
}

func MustParse(t string) *template.Template {
	tmpl, err := New().Parse(t)
	if err != nil {
		log.Fatal(err)
	}
	return tmpl
}

func MustParseFile(t string) *template.Template {
	data, err := os.ReadFile(t)
	if err != nil {
		log.Fatal(err)
	}
	tmpl, err := New().Parse(string(data))
	if err != nil {
		log.Fatal(err)
	}
	return tmpl
}

func Render(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	s := string(b)
	tmpl, err := New().Parse(s)
	if err != nil {
		return "", err
	}
	buf := new(bytes.Buffer)
	if err = tmpl.Execute(buf, nil); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func RenderDelims(path, right, left string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	s := string(b)

	tmpl, err := New().Delims(right, left).Parse(s)
	if err != nil {
		return "", err
	}
	buf := new(bytes.Buffer)
	if err = tmpl.Execute(buf, nil); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func tplFun(parent *template.Template) func(string) (string, error) {
	return func(tpl string) (string, error) {
		if strings.TrimSpace(tpl) == "" {
			// Empty templates will give us a stack trace
			return "", nil
		}
		t, err := parent.Clone()
		if err != nil {
			return "", fmt.Errorf("%w: cannot clone template", err)
		}
		t, err = t.Parse(tpl)
		if err != nil {
			return "", fmt.Errorf("%w: cannot parse template", err)
		}

		var buf strings.Builder
		if err = t.Execute(&buf, nil); err != nil {
			return "", fmt.Errorf("%w: error during tpl function execution for %q", err, tpl)
		}

		// See comment in renderWithReferences explaining the <no value> hack.
		return strings.ReplaceAll(buf.String(), "<no value>", ""), nil
	}
}
