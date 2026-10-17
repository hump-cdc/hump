package config

import (
	"fmt"

	"github.com/goccy/go-yaml"
)

// expand expands a configuration YAML by:
//   - flattening every `tables[].tables` group into one entry per nested table,
//     copying the group's other properties (e.g. sinks) onto each;
//   - merging entries that share the same schema/table after flattening;
//   - resolving `preset: <name>` references in sinks against the top-level
//     `sinks` section;
//   - pushing a table's `operations` down onto each of its sinks, so the
//     list survives the merge below;
//   - dropping the top-level `sinks` section from the output.
func expand(raw string) (string, error) {
	var root any
	if err := yaml.UnmarshalWithOptions([]byte(raw), &root, yaml.UseOrderedMap()); err != nil {
		return "", err
	}

	rootMap, ok := root.(yaml.MapSlice)
	if !ok {
		return raw, nil
	}

	presets := collectSinkPresets(rootMap)

	flat := flattenTables(rootMap)

	if err := resolveSinkPresets(flat, presets); err != nil {
		return "", err
	}

	pushDownOperations(flat)

	merged := mergeTablesByKey(flat)

	result := yaml.MapSlice{}
	for _, item := range rootMap {
		key, _ := item.Key.(string)
		if key == "sinks" || key == "tables" {
			continue
		}
		result = append(result, item)
	}

	tablesResult := make([]any, len(merged))
	for i, t := range merged {
		tablesResult[i] = t
	}
	result = append(result, yaml.MapItem{Key: "tables", Value: tablesResult})

	return canonicalYAML(result)
}

// collectSinkPresets builds a name -> sink config map from the top-level
// `sinks` section. The `name` key is stripped from each stored entry.
func collectSinkPresets(root yaml.MapSlice) map[string]yaml.MapSlice {
	out := map[string]yaml.MapSlice{}

	sinksRaw, ok := mapGet(root, "sinks")
	if !ok {
		return out
	}

	sinks, _ := sinksRaw.([]any)
	for _, s := range sinks {
		sm, ok := s.(yaml.MapSlice)
		if !ok {
			continue
		}

		nameAny, ok := mapGet(sm, "name")
		if !ok {
			continue
		}

		name, _ := nameAny.(string)
		out[name] = mapDelete(sm, "name")
	}

	return out
}

// flattenTables expands every `tables[].tables` group into one entry per
// nested table, carrying the group's other properties onto each child.
// Entries that already use the `table`/`schema` form pass through unchanged.
func flattenTables(root yaml.MapSlice) []yaml.MapSlice {
	tablesRaw, ok := mapGet(root, "tables")
	if !ok {
		return nil
	}

	list, _ := tablesRaw.([]any)
	flat := make([]yaml.MapSlice, 0, len(list))
	for _, t := range list {
		tm, ok := t.(yaml.MapSlice)
		if !ok {
			continue
		}

		nested, hasNested := mapGet(tm, "tables")
		if !hasNested {
			flat = append(flat, tm)
			continue
		}

		nestedList, _ := nested.([]any)
		shared := mapDelete(tm, "tables")
		for _, n := range nestedList {
			nm, ok := n.(yaml.MapSlice)
			if !ok {
				continue
			}
			flat = append(flat, mergeMapSlice(nm, shared))
		}
	}

	return flat
}

// resolveSinkPresets replaces every `preset: <name>` sink entry with the
// referenced preset's body, layering any sibling keys on top.
func resolveSinkPresets(tables []yaml.MapSlice, presets map[string]yaml.MapSlice) error {
	for i, t := range tables {
		sinksRaw, ok := mapGet(t, "sinks")
		if !ok {
			continue
		}

		list, _ := sinksRaw.([]any)
		resolved := make([]any, 0, len(list))
		for _, sk := range list {
			sm, ok := sk.(yaml.MapSlice)
			if !ok {
				resolved = append(resolved, sk)
				continue
			}

			presetRaw, hasPreset := mapGet(sm, "preset")
			if !hasPreset {
				resolved = append(resolved, sm)
				continue
			}

			presetName, _ := presetRaw.(string)
			base, exists := presets[presetName]
			if !exists {
				return fmt.Errorf("unknown sink preset: %q", presetName)
			}

			rest := mapDelete(sm, "preset")
			resolved = append(resolved, mergeMapSlice(rest, base))
		}

		tables[i] = mapSet(t, "sinks", resolved)
	}

	return nil
}

// pushDownOperations moves a table's `operations` onto every one of its sinks,
// overriding whatever the sink or its preset set, and drops it from the table.
// A table without `operations` leaves its sinks' own lists untouched. The list
// has to live on the sinks: merging keeps only the first entry's table keys, so
// a table-level list would be lost for the sinks of every later entry.
func pushDownOperations(tables []yaml.MapSlice) {
	for i, t := range tables {
		ops, ok := mapGet(t, "operations")
		if !ok {
			continue
		}

		t = mapDelete(t, "operations")

		if sinksRaw, ok := mapGet(t, "sinks"); ok {
			list, _ := sinksRaw.([]any)
			sinks := make([]any, 0, len(list))
			for _, sk := range list {
				if sm, ok := sk.(yaml.MapSlice); ok {
					sk = mapSet(sm, "operations", ops)
				}
				sinks = append(sinks, sk)
			}
			t = mapSet(t, "sinks", sinks)
		}

		tables[i] = t
	}
}

// mergeTablesByKey collapses entries that share the same schema/table pair.
// Non-sink keys take the first entry's value; sinks are concatenated and
// deduplicated using a YAML-canonical form of each entry.
func mergeTablesByKey(tables []yaml.MapSlice) []yaml.MapSlice {
	type key struct{ schema, table string }

	idx := map[key]int{}
	out := make([]yaml.MapSlice, 0, len(tables))
	for _, t := range tables {
		schemaRaw, _ := mapGet(t, "schema")
		tableRaw, _ := mapGet(t, "table")
		schema, _ := schemaRaw.(string)
		table, _ := tableRaw.(string)
		k := key{schema, table}

		if i, found := idx[k]; found {
			merged, err := mergeTableEntries(out[i], t)
			if err == nil {
				out[i] = merged
			}
			continue
		}

		idx[k] = len(out)
		out = append(out, t)
	}

	return out
}

// mergeTableEntries combines two entries that share schema/table. Non-sink
// keys keep `a`'s value; sinks are concatenated (a then b) and deduplicated.
func mergeTableEntries(a, b yaml.MapSlice) (yaml.MapSlice, error) {
	merged := mergeMapSlice(a, b)

	aSinks, _ := mapGet(a, "sinks")
	bSinks, _ := mapGet(b, "sinks")

	aList, _ := aSinks.([]any)
	bList, _ := bSinks.([]any)

	if len(aList) == 0 && len(bList) == 0 {
		return merged, nil
	}

	combined := make([]any, 0, len(aList)+len(bList))
	seen := map[string]struct{}{}
	for _, s := range aList {
		k, err := canonicalYAML(s)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		combined = append(combined, s)
	}

	for _, s := range bList {
		k, err := canonicalYAML(s)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		combined = append(combined, s)
	}

	return mapSet(merged, "sinks", combined), nil
}

// canonicalYAML marshals v to a YAML string; equal values yield equal output,
// so it also serves as a deep-equality key.
func canonicalYAML(v any) (string, error) {
	buf, err := yaml.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

// mergeMapSlice returns a + b where a's value wins on key conflicts and a's
// key order is preserved, with b's extra keys appended in b's order.
func mergeMapSlice(a, b yaml.MapSlice) yaml.MapSlice {
	out := make(yaml.MapSlice, 0, len(a)+len(b))
	seen := make(map[any]struct{}, len(a))
	for _, kv := range a {
		out = append(out, kv)
		seen[kv.Key] = struct{}{}
	}
	for _, kv := range b {
		if _, exists := seen[kv.Key]; exists {
			continue
		}
		out = append(out, kv)
	}
	return out
}

// mapGet returns the value bound to key in ms, or (nil, false) if absent.
func mapGet(ms yaml.MapSlice, key string) (any, bool) {
	for _, kv := range ms {
		if k, ok := kv.Key.(string); ok && k == key {
			return kv.Value, true
		}
	}
	return nil, false
}

// mapSet returns a copy of ms with key set to value, preserving the original
// key position when present and appending otherwise.
func mapSet(ms yaml.MapSlice, key string, value any) yaml.MapSlice {
	out := make(yaml.MapSlice, 0, len(ms)+1)
	found := false
	for _, kv := range ms {
		if k, ok := kv.Key.(string); ok && k == key {
			out = append(out, yaml.MapItem{Key: key, Value: value})
			found = true
			continue
		}
		out = append(out, kv)
	}
	if !found {
		out = append(out, yaml.MapItem{Key: key, Value: value})
	}
	return out
}

// mapDelete returns a copy of ms without any entry whose key equals key.
func mapDelete(ms yaml.MapSlice, key string) yaml.MapSlice {
	out := make(yaml.MapSlice, 0, len(ms))
	for _, kv := range ms {
		if k, ok := kv.Key.(string); ok && k == key {
			continue
		}
		out = append(out, kv)
	}
	return out
}
