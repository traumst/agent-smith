package ui

import (
	"embed"
	"html/template"
)

//go:embed static/*
var StaticFS embed.FS

//go:embed static/templates/*
var TemplateFS embed.FS

// LoadTemplates parses all HTML templates from the embedded filesystem.
func LoadTemplates() (*template.Template, error) {
	return template.ParseFS(TemplateFS, "static/templates/*.html")
}
