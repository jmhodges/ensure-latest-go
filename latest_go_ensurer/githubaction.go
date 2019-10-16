package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

func updateGitHubActionFiles(actionfilePaths map[string]bool, goVers string) ([]fileContent, error) {
	var files []fileContent
	for fp, _ := range actionfilePaths {
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

		contentsToWrite, err := updateSingleGitHubActionFile(fp, origFileContents, goVers)
		if err != nil {
			return nil, err
		}
		if contentsToWrite != nil {
			files = append(files, fileContent{origFP: fp, contentsToWrite: contentsToWrite})
		}
	}
	return files, nil
}

func updateSingleGitHubActionFile(fp string, origFileContents []byte, goVers string) ([]byte, error) {
	outer := make(map[string]interface{})
	err := yaml.Unmarshal(origFileContents, outer)
	if err != nil {
		return nil, fmt.Errorf("unable to parse GitHub Action config file %#v: %s", fp, err)
	}

	jobsInt, found := outer["jobs"]
	if !found {
		return origFileContents, nil
	}
	jobs, ok := jobsInt.(map[interface{}]interface{})
	if !ok {
		return origFileContents, nil
	}
	var fileContentsUpdated bool
	for _, jobInt := range jobs {
		job, ok := jobInt.(map[interface{}]interface{})
		if !ok {
			break
		}
		updated, err := updateGitHubActionGoMatrix(fp, job, goVers)
		if err != nil {
			return nil, err
		}
		if updated {
			fileContentsUpdated = true
		}
		stepsInt, found := job["steps"]
		if !found {
			continue
		}
		steps, ok := stepsInt.([]interface{})
		if !ok {
			continue
		}
		for i, stepInt := range steps {
			step, ok := stepInt.(map[interface{}]interface{})
			usesInt, found := step["uses"]
			if !found {
				continue
			}
			uses, ok := usesInt.(string)
			if !ok {
				continue
			}

			if strings.HasPrefix(uses, "actions/setup-go@") || uses == "actions/setup-go" {
				withInt, found := step["with"]
				if !found {
					step["with"] = map[string]string{
						"go-version": goVers,
					}
				}
				with, ok := withInt.(map[interface{}]interface{})
				if !ok {
					break
				}
				oldGoVersInt := with["go-version"]
				oldGoVers, ok := oldGoVersInt.(string)
				if !ok {
					return nil, fmt.Errorf("go-version in GitHub Action config file %#v wasn't a string as expected", fp)
				}

				isLiteral := strings.Index(oldGoVers, "{{") == -1
				if isLiteral && oldGoVers != goVers {
					with["go-version"] = goVers
					step["with"] = with
					fileContentsUpdated = true
				} else {
					// We're going to assume everyone uses {{ matrix.go }} as
					// the variable if go-version isn't a single version and so
					// do nothing here.
				}
				steps[i] = step
			}
		}
	}
	if !fileContentsUpdated {
		return origFileContents, nil
	}
	// Make sure that "name" and "on" show up in the file first, because that's
	// where most people put theirs and our yaml marshal re-orders the keys
	// alphabetically.
	b, err := topLevelOrderPreservedYaml(outer, origFileContents)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal modified GitHub Action object as YAML: %s", err)
	}
	return b, nil
}

func updateGitHubActionGoMatrix(fp string, job map[interface{}]interface{}, goVers string) (bool, error) {
	wrapped := job
	for _, key := range []string{"strategy", "matrix"} {
		wrappedInt, found := wrapped[key]
		if !found {
			return false, nil
		}
		wrapped2, ok := wrappedInt.(map[interface{}]interface{})
		if !ok {
			return false, nil
		}
		wrapped = wrapped2
	}
	var out []interface{}
	oldGoVersInt, found := wrapped["go"]
	if !found {
		return false, nil // No matrix.go to find
	}
	oldGoVers, ok := oldGoVersInt.([]interface{})
	if !ok {
		return false, fmt.Errorf("strategy.matrix.go wasn't an array as expected in GitHub Action config file %#v", fp)
	}
	versions := make(map[string]bool)
	for _, oldVersInt := range oldGoVers {
		oldVers, ok := oldVersInt.(string)
		if !ok {
			return false, fmt.Errorf("unknown type in 'matrix.go' array in GitHub Action config file %#v", fp)
		}
		if !versions[oldVers] {
			out = append(out, oldVers)
			versions[oldVers] = true
		}
	}
	var updated bool
	if !versions[goVers] {
		updated = true
		out = append(out, interface{}(goVers))
	}
	wrapped["go"] = out
	return updated, nil
}
