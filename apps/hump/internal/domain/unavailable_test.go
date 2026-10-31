package domain

import (
	"encoding/json"
	"testing"

	"hump/pkg/template"

	"github.com/goccy/go-yaml"
)

func TestUnavailableRendersAsMarker(t *testing.T) {
	if got := Unavailable.String(); got != "__hump_unavailable_value" {
		t.Errorf("String() = %q, want unavailable marker", got)
	}
	if Unavailable != (UnavailableValue{}) || !Unavailable.IsUnavailable() {
		t.Fatal("unavailable must remain comparable and identifiable")
	}

	got, err := json.Marshal(DbRecord{"blob": Unavailable, "empty": "", "null": nil})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if string(got) != `{"blob":"__hump_unavailable_value","empty":"","null":null}` {
		t.Errorf("json.Marshal() = %s, want distinct marker, empty string and null", got)
	}

	got, err = yaml.Marshal(DbRecord{"blob": Unavailable})
	if err != nil {
		t.Fatalf("yaml.Marshal() error = %v", err)
	}
	if string(got) != "blob: __hump_unavailable_value\n" {
		t.Errorf("yaml.Marshal() = %q, want unavailable marker", got)
	}
}

func TestUnavailableRendersThroughTemplates(t *testing.T) {
	data := map[string]any{"Next": DbRecord{"blob": Unavailable}}

	tests := []struct {
		name   string
		source string
		want   string
	}{
		{"field", "{{ .Next.blob }}", "__hump_unavailable_value"},
		{"json", "{{ .Next | json false }}", `{"blob":"__hump_unavailable_value"}`},
		{"yaml", "{{ .Next | yaml false }}", `blob: __hump_unavailable_value`},
		{"tostring", "{{ .Next.blob | tostring }}", "__hump_unavailable_value"},
		{"default", `{{ .Next.blob | default "n/a" }}`, "__hump_unavailable_value"},
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

func TestUnavailableRequiredFails(t *testing.T) {
	tmpl, err := template.New(`{{ required "missing blob" .Next.blob }}`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tmpl.Render(map[string]any{"Next": DbRecord{"blob": Unavailable}}); err == nil {
		t.Fatal("required must reject unavailable values")
	}
}
