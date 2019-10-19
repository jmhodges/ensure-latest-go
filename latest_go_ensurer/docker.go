package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
)

func updateDockerfiles(dockerfilePaths map[string]bool, goVers string) ([]fileContent, error) {
	var files []fileContent

	for fp, _ := range dockerfilePaths {
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

		contentsToWrite, err := updateSingleDockerfile(fp, origFileContents, goVers)
		if err != nil {
			return nil, err
		}
		if contentsToWrite != nil {
			files = append(files, fileContent{origFP: fp, contentsToWrite: contentsToWrite})
		}
	}
	return files, nil
}

func updateSingleDockerfile(fp string, origFileContents []byte, goVers string) ([]byte, error) {
	fileContents := make([]byte, len(origFileContents))
	copy(fileContents, origFileContents)
	lines := bytes.Split(fileContents, []byte{'\n'})
	for i, line := range lines {
		if bytes.HasPrefix(bytes.ToLower(bytes.TrimSpace(line)), []byte("from ")) {
			line, err := updateDockerfileFromLine(line, goVers)
			if err != nil {
				// This is almost certainly from a regexp compiliation problem,
				// but, just in case.
				return nil, fmt.Errorf("unable to attempt update todockerfile %v: %s", fp, err)
			}
			lines[i] = line
			break
		}
	}
	return bytes.Join(lines, []byte{'\n'}), nil
}

var dockerImageRe = regexp.MustCompile(`^(?P<prefix>(?i:from)\s+)golang(?P<tag>\:[\w-.]+)?(?P<suffix>(\s|#).*)?$`)
var dockerTagRe = regexp.MustCompile(`:\d+\.\d+(\.\d+)?-`)

func updateDockerfileFromLine(fromLine []byte, goVers string) ([]byte, error) {
	matches := dockerImageRe.FindSubmatch(fromLine)
	if len(matches) == 0 {
		return fromLine, nil
	}
	// Capturing the prefix preserves the capitalization and whitespace of the
	// FROM part.
	var prefix, suffix []byte
	for i, matchName := range dockerImageRe.SubexpNames() {
		if matchName == "prefix" {
			prefix = matches[i]
		}
		if matchName == "suffix" {
			suffix = matches[i]
		}
		if matchName == "tag" {
			// tag is :NUM.NUM.NUM-something
			tag := matches[i]
			if dockerTagRe.Match(tag) {
				ind := bytes.Index(tag, []byte{'-'})
				if ind != -1 {
					goVers += string(tag[ind:])
				}
			}
		}
	}
	newImage := "golang:" + goVers
	return append(prefix, append([]byte(newImage), suffix...)...), nil
}
