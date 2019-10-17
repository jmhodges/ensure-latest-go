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
	topMap := make(yaml.MapSlice, 0)
	err := yaml.Unmarshal(origFileContents, &topMap)
	if err != nil {
		return nil, fmt.Errorf("unable to parse GitHub Action config file %#v: %s", fp, err)
	}
	topMap = fixUpGitHubActionOnKey(fp, topMap)
	jobsInd, jobs, err := findMapItemAsMapSlice(topMap, "jobs")
	if err != nil {
		return nil, fmt.Errorf("GitHub Action config file %#v had an ununderstandable 'jobs' value", fp)
	}
	if jobsInd == -1 {
		return origFileContents, nil
	}
	newJobs, updated, err := updateGithubActionJobs(fp, jobs, goVers)
	if err != nil {
		return nil, err
	}
	if !updated {
		return origFileContents, nil
	}
	topMap[jobsInd] = yaml.MapItem{Key: "jobs", Value: newJobs}
	return yaml.Marshal(topMap)
}

func fixUpGitHubActionOnKey(fp string, topMap yaml.MapSlice) yaml.MapSlice {
	for tmi, mapItem := range topMap {
		boolK, ok := mapItem.Key.(bool)
		if ok && boolK {
			// gopkg.in/yaml.v2 is parsing GitHub Action's top-level "on" key as the boolean literal 'true'.
			// So, swap it out here. See https://github.com/go-yaml/yaml/issues/523
			topMap[tmi] = yaml.MapItem{Key: "on", Value: mapItem.Value}
			break
		}
	}
	return topMap
}

func updateGithubActionJobs(fp string, jobs yaml.MapSlice, goVers string) (yaml.MapSlice, bool, error) {
	var fileContentsUpdated bool
	for _, jobsMapItem := range jobs {
		jobName, ok := jobsMapItem.Key.(string)
		if !ok {
			return jobs, false, fmt.Errorf("non-string key inside jobs object YAML")
		}
		job, ok := jobsMapItem.Value.(yaml.MapSlice)
		if !ok {
			return jobs, false, fmt.Errorf("invalid type for job object in YAML")
		}
		_, steps, err := findMapItemAsMapSliceSlice(job, "steps")
		if err != nil {
			return jobs, false, fmt.Errorf("invalid 'steps' in job %#v object: %s", jobName, err)
		}
		for stepInd, step := range steps {
			usesInd, uses, err := findMapItemAsString(step, "uses")
			if err != nil {
				return jobs, false, fmt.Errorf("invalide 'uses' in 'steps' YAML object: %s", err)
			}
			if usesInd == -1 {
				continue
			}

			if strings.HasPrefix(uses, "actions/setup-go@") || uses == "actions/setup-go" {
				newStep, updated, err := updateGithubActionGoStep(step, goVers)
				if err != nil {
					return jobs, false, err
				}
				if updated {
					fileContentsUpdated = true
					steps[stepInd] = newStep
				}
			}
		}

		stratInd, strategy, err := findMapItemAsMapSlice(job, "strategy")
		if err != nil {
			return jobs, false, fmt.Errorf("invalid strategy object inside %#v job object YAML", jobName)
		}
		if stratInd != -1 {
			newStrategy, updated, err := updateGitHubActionGoMatrix(fp, strategy, goVers)
			if err != nil {
				return jobs, false, err
			}
			if updated {
				fileContentsUpdated = true
				job[stratInd] = yaml.MapItem{Key: "strategy", Value: newStrategy}
			}
		}
	}

	return jobs, fileContentsUpdated, nil
}

func updateGithubActionGoStep(step yaml.MapSlice, goVers string) (yaml.MapSlice, bool, error) {
	withInd, with, err := findMapItemAsMapSlice(step, "with")
	if err != nil {
		return step, false, fmt.Errorf("unable to find 'with' property in step YAML object: %s", err)
	}
	if withInd != -1 {
		oldGoVersInd, oldGoVers, err := findMapItemAsString(with, "go-version")
		if err != nil {
			return step, false, fmt.Errorf("unable to find 'go-version' property in 'with' YAML object: %s", err)
		}
		if oldGoVersInd != -1 {
			isLiteral := strings.Index(oldGoVers, "${{") == -1
			if isLiteral && oldGoVers != goVers {
				with[oldGoVersInd] = yaml.MapItem{Key: "go-version", Value: goVers}
				return step, true, nil
			} else {
				// We're going to assume if there's '{{' template markings, that
				// they're using {{ matrix.go }} as the variable.
			}
		} else {
			with = append(with, yaml.MapItem{Key: "go-version", Value: goVers})
			step[withInd].Value = with
			return step, true, nil
		}
	} else {
		with := yaml.MapItem{
			Key: "with",
			Value: yaml.MapSlice{
				yaml.MapItem{Key: "go-version", Value: goVers},
			},
		}
		step = append(step, with)
		return step, true, nil
	}
	return step, false, nil
}

func updateGitHubActionGoMatrix(fp string, strategy yaml.MapSlice, goVers string) (yaml.MapSlice, bool, error) {
	i, matrix, err := findMapItemAsMapSlice(strategy, "matrix")
	if err != nil {
		return strategy, false, fmt.Errorf("in strategy object, %s", err)
	}
	if i == -1 {
		return strategy, false, nil
	}
	i, oldGoVersions, err := findMapItemAsStringSlice(matrix, "go")
	if err != nil {
		return strategy, false, fmt.Errorf("error looking for 'go' key in strategy YAML object: %s", err)
	}
	if i == -1 {
		return strategy, false, nil
	}
	versions := make(map[string]bool)
	var newVersions []string
	for _, vers := range oldGoVersions {
		if !versions[vers] {
			newVersions = append(newVersions, vers)
			versions[vers] = true
		}
	}
	if versions[goVers] {
		return strategy, false, err
	}
	newVersions = append(newVersions, goVers)
	matrix[i] = yaml.MapItem{Key: "go", Value: newVersions}
	return strategy, true, err
}

func findMapItemAsMapSlice(obj yaml.MapSlice, desiredKey string) (int, yaml.MapSlice, error) {
	i, out, err := findMapItem(obj, desiredKey)
	if err != nil {
		return i, yaml.MapSlice{}, err
	}
	if i == -1 {
		return i, yaml.MapSlice{}, err
	}
	ret, ok := out.(yaml.MapSlice)
	if !ok {
		return i, yaml.MapSlice{}, fmt.Errorf("value of %#v in YAML object was not the expected yaml.MapSlice type", desiredKey)
	}
	return i, ret, nil

}

func findMapItemAsMapSliceSlice(obj yaml.MapSlice, desiredKey string) (int, []yaml.MapSlice, error) {
	i, mid, err := findMapItem(obj, desiredKey)
	if err != nil {
		return i, nil, err
	}
	if i == -1 {
		return i, nil, err
	}
	out, ok := mid.([]interface{})
	if !ok {
		return i, nil, fmt.Errorf("value of %#v in YAML object was not the expected []yaml.MapSlice type", desiredKey)
	}
	ret := make([]yaml.MapSlice, 0, len(out))
	for _, x := range out {
		str, ok := x.(yaml.MapSlice)
		if !ok {
			return i, nil, fmt.Errorf("value of %#v in YAML object was not the expected []yaml.MapSlice type", desiredKey)
		}
		ret = append(ret, str)
	}
	return i, ret, nil

}

func findMapItemAsStringSlice(obj yaml.MapSlice, desiredKey string) (int, []string, error) {
	i, mid, err := findMapItem(obj, desiredKey)
	if err != nil {
		return i, nil, err
	}
	if i == -1 {
		return i, nil, err
	}
	out, ok := mid.([]interface{})
	if !ok {
		return i, nil, fmt.Errorf("value of %#v in YAML object was not the expected []string type", desiredKey)
	}
	ret := make([]string, 0, len(out))
	for _, x := range out {
		str, ok := x.(string)
		if !ok {
			return i, nil, fmt.Errorf("value of %#v in YAML object was not the expected []string type", desiredKey)
		}
		ret = append(ret, str)
	}
	return i, ret, nil
}

func findMapItemAsString(obj yaml.MapSlice, desiredKey string) (int, string, error) {
	i, out, err := findMapItem(obj, desiredKey)
	if err != nil {
		return i, "", err
	}
	if i == -1 {
		return i, "", err
	}
	ret, ok := out.(string)
	if !ok {
		return i, "", fmt.Errorf("value of %#v in YAML object was not the expected string type", desiredKey)
	}
	return i, ret, nil

}

func findMapItem(obj yaml.MapSlice, desiredKey string) (int, interface{}, error) {
	for i, item := range obj {
		k, ok := item.Key.(string)
		if !ok {
			return -1, nil, fmt.Errorf("non-string key found in YAML object")
		}
		if k == desiredKey {
			return i, item.Value, nil
		}
	}
	return -1, nil, nil
}
