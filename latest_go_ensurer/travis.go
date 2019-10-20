package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"gopkg.in/laverya/yaml.v3"
)

func updateTravisFiles(travisfilePaths map[string]bool, goVers string) ([]fileContent, error) {
	var files []fileContent
	for fp, _ := range travisfilePaths {
		// O_RDWR so we can ensure we can write to the file without doing a
		// bunch of work first
		f, err := os.OpenFile(fp, os.O_RDWR, 0644)
		if err != nil {
			return nil, fmt.Errorf("unable to open Travis CI config file %#v for reading: %w", fp, err)
		}
		defer f.Close()
		origFileContents, err := ioutil.ReadAll(f)
		if err != nil {
			return nil, fmt.Errorf("unable to read contents of Travis CI config file %#v: %s", fp, err)
		}

		contentsToWrite, err := updateSingleTravisFile(fp, origFileContents, goVers)
		if err != nil {
			return nil, fmt.Errorf("unable to parse YAML Travis CI config file %#v: %s", fp, err)
		}
		if contentsToWrite != nil {
			files = append(files, fileContent{origFP: fp, contentsToWrite: contentsToWrite})
		}
	}
	return files, nil
}

func updateSingleTravisFile(fp string, origFileContents []byte, goVers string) ([]byte, error) {
	ty := &yaml.Node{}
	err := yaml.Unmarshal(origFileContents, ty)
	if err != nil {
		return nil, err
	}

	node, err := findYAMLObject(ty, "go")
	if err != nil {
		return nil, err
	}
	if node == nil {
		return origFileContents, nil
	}
	var fileContentsUpdated bool
	switch node.Kind {
	case yaml.ScalarNode:
		oldGoVers := node.Value
		switch node.Tag {
		case "!!str", "!!int":
			// oldGoVers is fine
		default:
			return nil, fmt.Errorf("unsupported type for 'go' value in travis config file %#v. Must be a string, or sequence", fp)
		}
		if oldGoVers != goVers {
			node.Value = goVers
			fileContentsUpdated = true
		}
	case yaml.SequenceNode:
		versions := make(map[string]bool)
		var out []*yaml.Node

		for _, child := range node.Content {
			if child.Kind != yaml.ScalarNode {
				return nil, fmt.Errorf("'go' value in Travis CI config file was not a sequence of strings or ints")
			}
			oldGoVers := child.Value
			if !versions[oldGoVers] {
				out = append(out, child)
				versions[oldGoVers] = true
			}
		}
		if !versions[goVers] {
			fileContentsUpdated = true
			if len(versions) == 1 {
				// If it's just one version in the original file, swap it out
				// whole cloth.
				log.Println("original node was", node.Kind, node.Content)
				node.Content = []*yaml.Node{
					&yaml.Node{
						Kind:  yaml.ScalarNode,
						Tag:   "!!str",
						Value: goVers,
					},
				}
			} else {
				node.Content = append(out,
					&yaml.Node{
						Kind:  yaml.ScalarNode,
						Tag:   "!!str",
						Value: goVers,
					},
				)
			}
		}
	default:
		return nil, fmt.Errorf("unknown type for 'go' value in travis config file %#v: %s", fp, err)
	}
	if fileContentsUpdated {
		return yamlMarshal(ty)
	}
	return origFileContents, nil
}
