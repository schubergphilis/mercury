package web

import (
	"errors"
	"fmt"
	"html/template"
	"time"

	"github.com/schubergphilis/mercury/pkg/tlsconfig"

	rice "github.com/GeertJohan/go.rice"
)

// Config for WEB
type Config struct {
	Binding   string              `toml:"binding" json:"binding"`
	Port      int                 `toml:"port" json:"port"`
	Path      string              `toml:"path" json:"path"`
	TLSConfig tlsconfig.TLSConfig `json:"tls" toml:"tls" yaml:"tls"`
}

// Page data
type Page struct {
	Title    string
	URI      string
	Hostname string
	Time     time.Time
}

// LoadTemplates a template from the rice box
func LoadTemplates(box string, name []string) (*template.Template, error) {
	templateBox, err := rice.FindBox(box)
	if err != nil {
		return nil, fmt.Errorf("Error loading box:%s error:%s", box, err.Error())
	}

	var combined string
	for _, templateName := range name {
		templateString, erro := templateBox.String(templateName)
		if erro != nil {
			return nil, fmt.Errorf("Error loading file:%s error: %s", templateName, erro.Error())
		}
		combined = combined + templateString
	}

	tmplMessage, err := template.New(name[0]).Funcs(template.FuncMap{
		"dict": func(values ...interface{}) (map[string]interface{}, error) {
			if len(values)%2 != 0 {
				return nil, errors.New("invalid dict call")
			}

			dict := make(map[string]interface{}, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, errors.New("dict keys must be strings")
				}

				dict[key] = values[i+1]
			}
			return dict, nil
		},
	}).Parse(combined)

	if err != nil {
		return nil, err
	}

	return tmplMessage, nil
}
