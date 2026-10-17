package sink

import (
	"hump/config/decoder"
	sinkport "hump/internal/port/sink"

	"github.com/goccy/go-yaml/ast"
)

// getSinkType returns the single sink discriminator present in node along with
// its value node. Returns ("", nil) if none or more than one is present.
func getSinkType(node *ast.MappingNode) (sinkport.Type, ast.Node) {
	var foundType sinkport.Type
	var foundValue ast.Node

	for _, t := range sinkport.SinkTypes {
		v := decoder.KeyValue(node, string(t))
		if v == nil {
			continue
		}

		if foundType != "" {
			return "", nil
		}
		foundType = t
		foundValue = v
	}

	return foundType, foundValue
}

// nodeWithoutSinkTypes returns a shallow clone of m with any entry
// whose key matches a known sink type removed. The original node is left intact.
func nodeWithoutSinkTypes(m *ast.MappingNode) *ast.MappingNode {
	skip := make(map[string]struct{}, len(sinkport.SinkTypes))
	for _, t := range sinkport.SinkTypes {
		skip[string(t)] = struct{}{}
	}

	filtered := make([]*ast.MappingValueNode, 0, len(m.Values))
	for _, kv := range m.Values {
		if _, drop := skip[kv.Key.String()]; drop {
			continue
		}
		filtered = append(filtered, kv)
	}

	clone := *m
	clone.Values = filtered

	return &clone
}
