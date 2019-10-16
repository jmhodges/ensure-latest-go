package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

var (
	excludeFiles     = flag.String("excludeFiles", "", "List of comma-seperated file paths to not update to the latest Go")
	dockerfilesInput = flag.String("dockerfiles", "", "List of comma-separate file paths to Dockerfiles that aren't named 'Dockerfile' and somewhere in the current directory or below")
)

func main() {
	flag.Parse()

	excluded := make(map[string]bool)
	for _, ef := range strings.Split(*excludeFiles, ",") {
		ef := abs(ef)
		excluded[ef] = true
	}
	dockerfiles := make(map[string]bool)
	for _, df := range strings.Split(*dockerfilesInput, ",") {
		df = abs(df)
		if df == "" || excluded[df] {
			continue
		}
		dockerfiles[df] = true
	}

	var actionfiles, travisfiles []string
	if len(dockerfiles)+len(actionfiles)+len(travisfiles) == 0 {
		log.Fatalf("latest_go_ensurer: no files given to update. Set -dockerfiles, -travisfiles, or -githubActionFiles")
	}

	goVers, err := getLatestGoVersion()
	if err != nil {
		log.Fatalf("latest_go_ensurer: %s", err)
	}

	// Check that we can read and parse all of the files before writing changes
	// back to the file system. This won't avoid all partial write problems, but
	// it'll avoid obvious stuff.
	contents, err := updatedDockerfiles(dockerfiles, goVers)
	if err != nil {
		log.Fatalf("latest_go_ensurer: %s", err)
	}
	sort.Slice(contents, func(i, j int) bool {
		return contents[i].origFP < contents[j].origFP
	})
	for _, fc := range contents {
		err := ioutil.WriteFile(fc.origFP, fc.contentsToWrite, 0644)
		if err != nil {
			log.Fatalf("latest_go_ensurer: unable to write new updated contents to %#v: %s", fc.origFP, err)
		}
	}
	log.Println(dockerfiles)
	for _, c := range contents {
		log.Println(string(c.contentsToWrite))
	}
	// var temps []string
	// for _, fc := range contents {

	// }
}

type fileContent struct {
	origFP          string
	contentsToWrite []byte
}

func updatedDockerfiles(dockerfilePaths map[string]bool, goVers string) ([]fileContent, error) {
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

		res, err := parser.Parse(bytes.NewBuffer(origFileContents))
		if err != nil {
			return nil, fmt.Errorf("unable to parse Dockerfile %#v: %s", fp, err)
		}
		if res.AST != nil && len(res.AST.Children) == 0 {
			return nil, fmt.Errorf("Dockerfile %#v seems to contain no lines of code", fp)
		}
		contentsToWrite, err := updateDockerfile(fp, origFileContents, res.AST, goVers)
		if err != nil {
			return nil, err
		}
		if contentsToWrite != nil {
			files = append(files, fileContent{origFP: fp, contentsToWrite: contentsToWrite})
		}
	}
	return files, nil
}

func updateDockerfile(fp string, origFileContents []byte, orig *parser.Node, goVers string) ([]byte, error) {
	lines := bytes.Split(origFileContents, []byte{'\n'})
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

func getLatestGoVersion() (string, error) {
	// FIXME implement
	return "1.1", nil
}

func abs(fp string) string {
	out, err := filepath.Abs(filepath.Clean(fp))
	if err != nil {
		log.Fatalf("latest_go_ensurer: unable to get absolute path of %#v: %s", fp, err)
	}
	return out
}
