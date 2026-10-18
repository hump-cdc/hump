package domain

import (
	"encoding/json"
	"testing"

	"hump/pkg/template"

	"github.com/goccy/go-yaml"
)

func TestUnavailableRendersAsEmptyString(t *testing.T) {
	if got := Unavailable.String(); got != "" {
		t.Errorf("String() = %q, want an empty string", got)
	}

	got, err := json.Marshal(DbRecord{"blob": Unavailable})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if string(got) != `{"blob":""}` {
		t.Errorf(`json.Marshal() = %s, want {"blob":""}`, got)
	}

	got, err = yaml.Marshal(DbRecord{"blob": Unavailable})
	if err != nil {
		t.Fatalf("yaml.Marshal() error = %v", err)
	}
	if string(got) != "blob: \"\"\n" {
		t.Errorf(`yaml.Marshal() = %q, want "blob: \"\"\n"`, got)
	}
}

func TestUnavailableRendersThroughTemplates(t *testing.T) {
	data := map[string]any{"Next": DbRecord{"blob": Unavailable}}

	tests := []struct {
		name   string
		source string
		want   string
	}{
		{"field", "{{ .Next.blob }}", ""},
		{"json", "{{ .Next | json false }}", `{"blob":""}`},
		{"yaml", "{{ .Next | yaml false }}", `blob: ""`},
		{"tostring", "{{ .Next.blob | tostring }}", ""},
		{"default", `{{ .Next.blob | default "n/a" }}`, ""},
		{"empty", "{{ empty .Next.blob }}", "true"},
		{"eq", `{{ eq .Next.blob "" }}`, "true"},
		{"ne", `{{ ne .Next.blob "set" }}`, "true"},
		{"coalesce", `{{ coalesce .Next.blob "fallback" }}`, "fallback"},
		{"unavailable", "{{ if unavailable .Next.blob }}missing{{ end }}", "missing"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := template.New(tt.source)
			if err != nil {
				t.Fatalf("template.New(%q) error = %v", tt.source, err)
			}

			got, err := tmpl.Render(data)
			if err != nil {
				t.Fatalf("Render() error = %v", err)
			}

			if got != tt.want {
				t.Errorf("Render() = %q, want %q", got, tt.want)
			}
		})
	}
}
