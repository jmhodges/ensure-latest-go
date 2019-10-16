package main

import (
	"bytes"
	"fmt"
	"regexp"

	"gopkg.in/yaml.v2"
)

const regexpPrefixName = "key"

var keyPrefixRE = regexp.MustCompile(`^(?P<` + regexpPrefixName + `>[\w-.]+):`)

func topLevelOrderPreservedYaml(obj map[string]interface{}, origFileContents []byte) ([]byte, error) {
	lines := bytes.Split(origFileContents, []byte{'\n'})
	unorderedBytes := make(map[string][]byte)
	for k, v := range obj {
		b, err := yaml.Marshal(map[string]interface{}{k: v})
		if err != nil {
			return nil, fmt.Errorf("unable to re-marshal YAML object with key %#v and body %s: %s", k, v, err)
		}
		unorderedBytes[k] = b
	}
	var orderedKeys []string
	for _, line := range lines {
		matches := keyPrefixRE.FindSubmatch(line)
		for i, name := range keyPrefixRE.SubexpNames() {
			if name == regexpPrefixName && len(matches) != 0 && len(matches[i]) != 0 {
				out := bytes.Trim(matches[i], `"`)
				orderedKeys = append(orderedKeys, string(out))
			}
		}
	}
	buf := &bytes.Buffer{}
	for _, key := range orderedKeys {
		b, found := unorderedBytes[key]
		if !found {
			return nil, fmt.Errorf("expected to find a key %#v for YAML marshalling, but didn't find it in the given object", key)
		}
		buf.Write(b)
	}
	return buf.Bytes(), nil
}
