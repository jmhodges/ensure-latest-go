package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"gopkg.in/yaml.v2"
)

func updateTravisFiles(travisfilePaths map[string]bool, goVers string) ([]fileContent, error) {
	var files []fileContent
	for fp, _ := range travisfilePaths {
		// O_RDWR so we can ensure we can write to the file without doing a
		// bunch of work first
		f, err := os.OpenFile(fp, os.O_RDWR, 0644)
		if err != nil {
			return nil, fmt.Errorf("unable to open Dockerfile %#v for reading: %w", fp, err)
		}
		defer f.Close()
		origFileContents, err := ioutil.ReadAll(f)
		if err != nil {
			return nil, fmt.Errorf("unable to read contents of Dockerfile %#v: %s", fp, err)
		}

		contentsToWrite, err := updateSingleTravisFile(fp, origFileContents, goVers)
		if err != nil {
			return nil, err
		}
		if contentsToWrite != nil {
			files = append(files, fileContent{origFP: fp, contentsToWrite: contentsToWrite})
		}
	}
	return files, nil
}

func updateSingleTravisFile(fp string, origFileContents []byte, goVers string) ([]byte, error) {
	ty := make(map[string]interface{})
	err := yaml.Unmarshal(origFileContents, ty)
	if err != nil {
		return nil, fmt.Errorf("unable to parse YAML travis config file %#v: %s", fp, err)
	}
	goVersions, found := ty["go"]
	if !found {
		return origFileContents, nil
	}
	var out []interface{}
	switch oldGoVers := goVersions.(type) {
	case string:
		if oldGoVers != goVers {
			out = []interface{}{oldGoVers, goVers}
		}
	case []interface{}:
		versions := make(map[string]bool)
		for _, oldVersInt := range oldGoVers {
			oldVers, ok := oldVersInt.(string)
			if !ok {
				return nil, fmt.Errorf("unknown type in 'go' array in travis config file %#v: %s", fp, err)
			}
			if !versions[oldVers] {
				out = append(out, oldVers)
				versions[oldVers] = true
			}
		}
		if !versions[goVers] {
			out = append(out, interface{}(goVers))
		}
	default:
		return nil, fmt.Errorf("unknown type for 'go' value in travis config file %#v: %s", fp, err)
	}
	log.Println("FIXMES out", out)
	ty["go"] = out
	b, err := yaml.Marshal(ty)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal YAML travis config file %#v: %s", fp, err)
	}
	return b, nil
}
