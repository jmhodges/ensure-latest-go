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

		contentsToWrite, err := updateDockerfile(fp, origFileContents, goVers)
		if err != nil {
			return nil, err
		}
		if contentsToWrite != nil {
			files = append(files, fileContent{origFP: fp, contentsToWrite: contentsToWrite})
		}
	}
	return files, nil
}

func updateDockerfile(fp string, origFileContents []byte, goVers string) ([]byte, error) {
	fileContents := make([]byte, len(origFileContents))
	copy(fileContents, origFileContents)
	lines := bytes.Split(fileContents, []byte{'\n'})
	for i, line := range lines {
		if bytes.HasPrefix(bytes.ToLower(bytes.TrimSpace(line)), []byte("from ")) {
			line, err := updateDockerfileFromLine(line, "golang", goVers)
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

func updateDockerfileFromLine(fromLine []byte, origImageName string, goVers string) ([]byte, error) {
	newImage := origImageName + ":" + goVers
	fromExpr := `^(?P<prefix>(?i:from)\s+)` + regexp.QuoteMeta(origImageName) + `(\:[\w-.]+)?(?P<suffix>(\s|#).*)?$`
	imageRe, err := regexp.Compile(fromExpr)
	if err != nil {
		return nil, fmt.Errorf("unable to compile regexp %#v for image %#v: %s", fromExpr, origImageName, err)
	}
	matches := imageRe.FindSubmatch(fromLine)
	if len(matches) == 0 {
		return fromLine, nil
	}
	// Capturing the prefix preserves the capitalization and whitespace of the
	// FROM part.
	var prefix, suffix []byte
	for i, matchName := range imageRe.SubexpNames() {
		if matchName == "prefix" {
			prefix = matches[i]
		}
		if matchName == "suffix" {
			suffix = matches[i]
		}
	}
	return append(prefix, append([]byte(newImage), suffix...)...), nil
}
