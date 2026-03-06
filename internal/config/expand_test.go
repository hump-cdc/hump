package config

import (
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
)

// normalizeYAML parses YAML and re-marshals it to obtain a canonical form
// suitable for comparing two YAML documents that may differ only in
// whitespace or key ordering of unrelated structures.
func normalizeYAML(t *testing.T, src string) string {
	t.Helper()
	var v any
	if err := yaml.UnmarshalWithOptions([]byte(src), &v, yaml.UseOrderedMap()); err != nil {
		t.Fatalf("normalize: %v", err)
	}
	out, err := yaml.Marshal(v)
	if err != nil {
		t.Fatalf("normalize marshal: %v", err)
	}
	return string(out)
}

func TestExpand(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "non-map root is returned unchanged",
			in:   `- a` + "\n" + `- b` + "\n",
			want: `- a` + "\n" + `- b` + "\n",
		},
		{
			name: "flat tables pass through and sinks section is dropped",
			in: `version: 1
sinks:
  - name: p1
    type: http
    url: http://x
tables:
  - schema: s
    table: t
    sinks:
      - type: http
        url: http://y
`,
			want: `version: 1
tables:
  - schema: s
    table: t
    sinks:
      - type: http
        url: http://y
`,
		},
		{
			name: "group with nested tables is flattened and shared properties propagate",
			in: `tables:
  - sinks:
      - type: http
        url: http://x
    tables:
      - schema: s1
        table: t1
      - schema: s1
        table: t2
`,
			want: `tables:
  - schema: s1
    table: t1
    sinks:
      - type: http
        url: http://x
  - schema: s1
    table: t2
    sinks:
      - type: http
        url: http://x
`,
		},
		{
			name: "preset reference is expanded with sibling override",
			in: `sinks:
  - name: kafka1
    type: kafka
    brokers: [a, b]
    topic: orig
tables:
  - schema: s
    table: t
    sinks:
      - preset: kafka1
        topic: override
`,
			want: `tables:
  - schema: s
    table: t
    sinks:
      - topic: override
        type: kafka
        brokers:
          - a
          - b
`,
		},
		{
			name: "duplicate schema/table entries are merged and sinks deduplicated",
			in: `tables:
  - schema: s
    table: t
    sinks:
      - type: http
        url: http://x
  - schema: s
    table: t
    sinks:
      - type: http
        url: http://x
      - type: http
        url: http://y
`,
			want: `tables:
  - schema: s
    table: t
    sinks:
      - type: http
        url: http://x
      - type: http
        url: http://y
`,
		},
		{
			name: "non-map entries inside tables list are dropped",
			in: `tables:
  - schema: s
    table: t
  - "not-a-map"
`,
			want: `tables:
  - schema: s
    table: t
`,
		},
		{
			name: "preset without name in sinks list is ignored",
			in: `sinks:
  - type: http
    url: http://x
tables:
  - schema: s
    table: t
`,
			want: `tables:
  - schema: s
    table: t
`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := expand(tc.in)
			if err != nil {
				t.Fatalf("expand: %v", err)
			}
			gotN := normalizeYAML(t, got)
			wantN := normalizeYAML(t, tc.want)
			if gotN != wantN {
				t.Fatalf("expand mismatch\n--- got ---\n%s\n--- want ---\n%s", gotN, wantN)
			}
		})
	}
}

func TestExpand_InvalidYAML(t *testing.T) {
	_, err := expand("a: [unterminated\n")
	if err == nil {
		t.Fatalf("expected error on invalid YAML")
	}
}

func TestExpand_UnknownPreset(t *testing.T) {
	in := `tables:
  - schema: s
    table: t
    sinks:
      - preset: missing
`
	_, err := expand(in)
	if err == nil {
		t.Fatalf("expected error for unknown preset")
	}
	if !strings.Contains(err.Error(), "missing") {
		t.Fatalf("expected error to mention preset name, got %q", err)
	}
}

func TestExpand_PreservesTopLevelOrderAndExtraKeys(t *testing.T) {
	in := `version: 1
debug: true
sinks:
  - name: p
    type: http
    url: http://x
tables:
  - schema: s
    table: t
extra: keep-me
`
	out, err := expand(in)
	if err != nil {
		t.Fatalf("expand: %v", err)
	}
	if !strings.Contains(out, "version: 1") {
		t.Fatalf("expected version preserved, got:\n%s", out)
	}
	if !strings.Contains(out, "debug: true") {
		t.Fatalf("expected debug preserved, got:\n%s", out)
	}
	if !strings.Contains(out, "extra: keep-me") {
		t.Fatalf("expected extra preserved, got:\n%s", out)
	}
	if strings.Contains(out, "sinks:\n  - name: p") {
		t.Fatalf("expected top-level sinks dropped, got:\n%s", out)
	}
	// "tables:" must appear after the other top-level keys (it is re-appended).
	tablesIdx := strings.Index(out, "tables:")
	extraIdx := strings.Index(out, "extra:")
	if tablesIdx < extraIdx {
		t.Fatalf("expected tables to be appended after other keys, got:\n%s", out)
	}
}

func TestCollectSinkPresets(t *testing.T) {
	root := yaml.MapSlice{
		{Key: "sinks", Value: []any{
			yaml.MapSlice{
				{Key: "name", Value: "p1"},
				{Key: "type", Value: "http"},
			},
			yaml.MapSlice{
				{Key: "type", Value: "kafka"},
			},
			"not-a-map",
			yaml.MapSlice{
				{Key: "name", Value: "p2"},
				{Key: "topic", Value: "t"},
			},
		}},
	}

	got := collectSinkPresets(root)
	if len(got) != 2 {
		t.Fatalf("want 2 presets, got %d: %#v", len(got), got)
	}
	if _, ok := got["p1"]; !ok {
		t.Fatalf("want preset p1")
	}
	if _, ok := got["p2"]; !ok {
		t.Fatalf("want preset p2")
	}
	if _, exists := mapGet(got["p1"], "name"); exists {
		t.Fatalf("name should be stripped from stored preset")
	}
	if v, _ := mapGet(got["p1"], "type"); v != "http" {
		t.Fatalf("want type=http in p1, got %v", v)
	}
}

func TestCollectSinkPresets_NoSinksSection(t *testing.T) {
	got := collectSinkPresets(yaml.MapSlice{{Key: "tables", Value: []any{}}})
	if len(got) != 0 {
		t.Fatalf("want empty map, got %#v", got)
	}
}

func TestFlattenTables(t *testing.T) {
	root := yaml.MapSlice{
		{Key: "tables", Value: []any{
			yaml.MapSlice{
				{Key: "schema", Value: "s1"},
				{Key: "table", Value: "t1"},
			},
			yaml.MapSlice{
				{Key: "sinks", Value: []any{
					yaml.MapSlice{{Key: "type", Value: "http"}},
				}},
				{Key: "tables", Value: []any{
					yaml.MapSlice{
						{Key: "schema", Value: "s2"},
						{Key: "table", Value: "ta"},
					},
					yaml.MapSlice{
						{Key: "schema", Value: "s2"},
						{Key: "table", Value: "tb"},
					},
					"skip-non-map",
				}},
			},
		}},
	}

	flat := flattenTables(root)
	if len(flat) != 3 {
		t.Fatalf("want 3 flattened entries, got %d", len(flat))
	}

	if v, _ := mapGet(flat[0], "table"); v != "t1" {
		t.Fatalf("want first table t1, got %v", v)
	}

	for i, want := range []string{"ta", "tb"} {
		if v, _ := mapGet(flat[i+1], "table"); v != want {
			t.Fatalf("nested[%d]: want %s, got %v", i, want, v)
		}
		sinks, ok := mapGet(flat[i+1], "sinks")
		if !ok {
			t.Fatalf("nested[%d]: shared sinks not propagated", i)
		}
		if _, ok := sinks.([]any); !ok {
			t.Fatalf("nested[%d]: sinks not a list", i)
		}
	}
}

func TestFlattenTables_NoTablesKey(t *testing.T) {
	if got := flattenTables(yaml.MapSlice{{Key: "version", Value: 1}}); got != nil {
		t.Fatalf("want nil, got %#v", got)
	}
}

func TestResolveSinkPresets(t *testing.T) {
	presets := map[string]yaml.MapSlice{
		"p1": {
			{Key: "type", Value: "http"},
			{Key: "url", Value: "http://base"},
		},
	}
	tables := []yaml.MapSlice{
		{
			{Key: "schema", Value: "s"},
			{Key: "table", Value: "t"},
			{Key: "sinks", Value: []any{
				yaml.MapSlice{
					{Key: "preset", Value: "p1"},
					{Key: "url", Value: "http://override"},
				},
				yaml.MapSlice{
					{Key: "type", Value: "kafka"},
				},
				"scalar-passthrough",
			}},
		},
	}

	if err := resolveSinkPresets(tables, presets); err != nil {
		t.Fatalf("resolveSinkPresets: %v", err)
	}

	sinks, _ := mapGet(tables[0], "sinks")
	list, _ := sinks.([]any)
	if len(list) != 3 {
		t.Fatalf("want 3 sinks, got %d", len(list))
	}

	first, ok := list[0].(yaml.MapSlice)
	if !ok {
		t.Fatalf("first sink not a map")
	}
	if v, _ := mapGet(first, "type"); v != "http" {
		t.Fatalf("want type=http from preset, got %v", v)
	}
	if v, _ := mapGet(first, "url"); v != "http://override" {
		t.Fatalf("want sibling override to win, got %v", v)
	}
	if _, exists := mapGet(first, "preset"); exists {
		t.Fatalf("preset key should be removed after resolution")
	}

	if list[2] != "scalar-passthrough" {
		t.Fatalf("non-map sink should pass through")
	}
}

func TestResolveSinkPresets_UnknownPreset(t *testing.T) {
	tables := []yaml.MapSlice{
		{
			{Key: "sinks", Value: []any{
				yaml.MapSlice{{Key: "preset", Value: "ghost"}},
			}},
		},
	}
	err := resolveSinkPresets(tables, map[string]yaml.MapSlice{})
	if err == nil {
		t.Fatalf("expected error for unknown preset")
	}
}

func TestResolveSinkPresets_TableWithoutSinks(t *testing.T) {
	tables := []yaml.MapSlice{
		{{Key: "schema", Value: "s"}, {Key: "table", Value: "t"}},
	}
	if err := resolveSinkPresets(tables, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMergeTablesByKey(t *testing.T) {
	tables := []yaml.MapSlice{
		{
			{Key: "schema", Value: "s"},
			{Key: "table", Value: "t1"},
			{Key: "sinks", Value: []any{yaml.MapSlice{{Key: "type", Value: "http"}}}},
		},
		{
			{Key: "schema", Value: "s"},
			{Key: "table", Value: "t2"},
		},
		{
			{Key: "schema", Value: "s"},
			{Key: "table", Value: "t1"},
			{Key: "sinks", Value: []any{
				yaml.MapSlice{{Key: "type", Value: "http"}},
				yaml.MapSlice{{Key: "type", Value: "kafka"}},
			}},
		},
	}

	merged := mergeTablesByKey(tables)
	if len(merged) != 2 {
		t.Fatalf("want 2 merged entries, got %d", len(merged))
	}

	sinks, _ := mapGet(merged[0], "sinks")
	list, _ := sinks.([]any)
	if len(list) != 2 {
		t.Fatalf("want 2 deduped sinks on t1, got %d", len(list))
	}
}

func TestMergeTableEntries_NoSinksOnEither(t *testing.T) {
	a := yaml.MapSlice{{Key: "schema", Value: "s"}, {Key: "table", Value: "t"}}
	b := yaml.MapSlice{{Key: "table", Value: "t"}, {Key: "extra", Value: "v"}}

	got, err := mergeTableEntries(a, b)
	if err != nil {
		t.Fatalf("mergeTableEntries: %v", err)
	}
	if v, _ := mapGet(got, "extra"); v != "v" {
		t.Fatalf("want extra=v, got %v", v)
	}
	if _, ok := mapGet(got, "sinks"); ok {
		t.Fatalf("did not expect sinks key")
	}
}

func TestMergeMapSlice(t *testing.T) {
	a := yaml.MapSlice{
		{Key: "x", Value: 1},
		{Key: "y", Value: 2},
	}
	b := yaml.MapSlice{
		{Key: "y", Value: 99},
		{Key: "z", Value: 3},
	}

	out := mergeMapSlice(a, b)
	if len(out) != 3 {
		t.Fatalf("want 3 entries, got %d", len(out))
	}
	if v, _ := mapGet(out, "y"); v != 2 {
		t.Fatalf("want a's value to win for y, got %v", v)
	}
	if v, _ := mapGet(out, "z"); v != 3 {
		t.Fatalf("want z=3, got %v", v)
	}
	// key order: x, y from a, then z from b
	keys := make([]string, 0, len(out))
	for _, kv := range out {
		keys = append(keys, kv.Key.(string))
	}
	want := []string{"x", "y", "z"}
	for i, k := range want {
		if keys[i] != k {
			t.Fatalf("key order: want %v, got %v", want, keys)
		}
	}
}

func TestMapGet(t *testing.T) {
	ms := yaml.MapSlice{
		{Key: "a", Value: 1},
		{Key: 42, Value: "non-string-key"},
		{Key: "b", Value: "two"},
	}

	if v, ok := mapGet(ms, "b"); !ok || v != "two" {
		t.Fatalf("want two, got %v ok=%v", v, ok)
	}
	if _, ok := mapGet(ms, "missing"); ok {
		t.Fatalf("expected miss")
	}
}

func TestMapSet(t *testing.T) {
	t.Run("updates in place preserving order", func(t *testing.T) {
		ms := yaml.MapSlice{
			{Key: "a", Value: 1},
			{Key: "b", Value: 2},
			{Key: "c", Value: 3},
		}
		out := mapSet(ms, "b", 99)
		if len(out) != 3 {
			t.Fatalf("want 3, got %d", len(out))
		}
		if v, _ := mapGet(out, "b"); v != 99 {
			t.Fatalf("want b=99, got %v", v)
		}
		if out[1].Key != "b" {
			t.Fatalf("expected b to keep its position, got key %v at index 1", out[1].Key)
		}
	})

	t.Run("appends when missing", func(t *testing.T) {
		ms := yaml.MapSlice{{Key: "a", Value: 1}}
		out := mapSet(ms, "b", 2)
		if len(out) != 2 {
			t.Fatalf("want 2, got %d", len(out))
		}
		if out[1].Key != "b" {
			t.Fatalf("want b appended last, got %v", out[1].Key)
		}
	})
}

func TestMapDelete(t *testing.T) {
	ms := yaml.MapSlice{
		{Key: "a", Value: 1},
		{Key: "b", Value: 2},
		{Key: "c", Value: 3},
	}
	out := mapDelete(ms, "b")
	if len(out) != 2 {
		t.Fatalf("want 2, got %d", len(out))
	}
	if _, ok := mapGet(out, "b"); ok {
		t.Fatalf("expected b removed")
	}
	if _, ok := mapGet(out, "a"); !ok {
		t.Fatalf("expected a kept")
	}
	if _, ok := mapGet(out, "c"); !ok {
		t.Fatalf("expected c kept")
	}
}

func TestCanonicalYAML_MarshalError(t *testing.T) {
	// goccy cannot marshal channel values, so canonicalYAML surfaces the error.
	if _, err := canonicalYAML(make(chan int)); err == nil {
		t.Fatalf("expected marshal error for chan value")
	}
}

func TestMergeTableEntries_FirstSinkMarshalError(t *testing.T) {
	// An unmarshalable sink in the first entry fails deduplication of aList.
	a := yaml.MapSlice{
		{Key: "table", Value: "t"},
		{Key: "sinks", Value: []any{make(chan int)}},
	}
	b := yaml.MapSlice{{Key: "table", Value: "t"}}

	if _, err := mergeTableEntries(a, b); err == nil {
		t.Fatalf("expected error from unmarshalable sink in a")
	}
}

func TestMergeTableEntries_SecondSinkMarshalError(t *testing.T) {
	// A valid first entry lets aList complete; the bad sink in b fails bList.
	a := yaml.MapSlice{
		{Key: "table", Value: "t"},
		{Key: "sinks", Value: []any{yaml.MapSlice{{Key: "http", Value: "x"}}}},
	}
	b := yaml.MapSlice{
		{Key: "table", Value: "t"},
		{Key: "sinks", Value: []any{make(chan int)}},
	}

	if _, err := mergeTableEntries(a, b); err == nil {
		t.Fatalf("expected error from unmarshalable sink in b")
	}
}

func TestCanonicalYAML_StableForEquivalentMaps(t *testing.T) {
	a := yaml.MapSlice{{Key: "x", Value: 1}, {Key: "y", Value: 2}}
	b := yaml.MapSlice{{Key: "x", Value: 1}, {Key: "y", Value: 2}}

	ka, err := canonicalYAML(a)
	if err != nil {
		t.Fatalf("a: %v", err)
	}
	kb, err := canonicalYAML(b)
	if err != nil {
		t.Fatalf("b: %v", err)
	}
	if ka != kb {
		t.Fatalf("want equal canonical forms, got %q vs %q", ka, kb)
	}
}
