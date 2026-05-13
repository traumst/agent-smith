package ui

import (
	"embed"
	"html/template"
	"io/fs"
	"strings"
)

//go:embed static/*
var StaticFS embed.FS

//go:embed static/templates/*
var TemplateFS embed.FS

// LoadTemplates parses all HTML templates from the embedded filesystem.
// It returns a map of templates to avoid block name collisions.
func LoadTemplates() (map[string]*template.Template, error) {
	files, err := fs.ReadDir(TemplateFS, "static/templates")
	if err != nil {
		return nil, err
	}

	tmpls := make(map[string]*template.Template)
	for _, file := range files {
		name := file.Name()
		if file.IsDir() || name == "base.html" {
			continue
		}

		content, err := fs.ReadFile(TemplateFS, "static/templates/"+name)
		if err != nil {
			return nil, err
		}

		var t *template.Template
		// If it defines a content block, it's a full page that needs base.html
		if strings.Contains(string(content), `{{define "content"`) {
			// We parse base.html first so it is the main template for Execute()
			t, err = template.ParseFS(TemplateFS, "static/templates/base.html", "static/templates/"+name)
		} else {
			// Fragment
			t, err = template.ParseFS(TemplateFS, "static/templates/"+name)
		}

		if err != nil {
			return nil, err
		}
		tmpls[name] = t
	}
	return tmpls, nil
}
