package main

import (
	"bytes"
	"fmt"

	"gopkg.in/laverya/yaml.v3"
)

func findYAMLObject(obj *yaml.Node, desiredKey string) (*yaml.Node, error) {
	if obj.Kind == yaml.DocumentNode {
		return findYAMLObject(obj.Content[0], desiredKey)
	}
	if obj.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("YAML node is not a mapping node (was %d)", obj.Kind)
	}

	// YAML maps are reutrned as a MappingNode with Content. The even index
	// Nodes in Content are keys, and the odd index Nodes are the value of the
	// previous key.
	for i, item := range obj.Content {
		if i%2 != 0 {
			continue
		}
		if item.Kind != yaml.ScalarNode || item.Tag != "!!str" {
			return nil, fmt.Errorf("non-string key found in YAML object")
		}
		if item.Value == desiredKey {
			return obj.Content[i+1], nil
		}
	}
	return nil, nil
}

func yamlMarshal(obj *yaml.Node) ([]byte, error) {
	buf := &bytes.Buffer{}
	enc := yaml.NewEncoder(buf)
	enc.SetLineLength(-1) // Disable line wrap
	enc.SetIndent(2)
	err := enc.Encode(obj)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

/* FIXME delete
type mapSlice struct {
	node  *yaml.Node
	items []mapItem
}

func (ms *mapSlice) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("mapSlice can't be parsed from a %s, must be yaml.MappingNode", node.Kind)
	}
	ms.node = node
	for _, child := range node.Content {
		value := child.Contents[0]
		ms.items = append(ms.items, mapItem{
			node: child,
			Key: child.Value
		})
	}
	return nil
}

type mapItem struct {
	node  *yaml.Node
	Key   string
	Value *yaml.Node
}
*/
