package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func main() {
	excludeFiles = os.Getenv("INPUT_EXCLUDES")
	excluded := make(map[string]bool)
	for _, ef := range strings.Split(*excludeFiles, ",") {
		ef := abs(ef)
		excluded[ef] = true
	}

	dockerfilesInput = os.Getenv("INPUT_DOCKERFILES")
	dockerfiles := make(map[string]bool)
	for _, df := range strings.Split(*dockerfilesInput, ",") {
		df = abs(df)
		if df == "" || excluded[df] {
			continue
		}
		dockerfiles[df] = true
	}

	travisfilesInput = os.Getenv("INPUT_TRAVISFILES")
	travisfiles := make(map[string]bool)
	for _, tf := range strings.Split(*travisfilesInput, ",") {
		tf = abs(tf)
		if tf == "" || excluded[tf] {
			continue
		}
		travisfiles[tf] = true
	}

	actionfilesInput = os.Getenv("INPUT_ACTIONFILES")
	actionfiles := make(map[string]bool)
	for _, af := range strings.Split(*actionfilesInput, ",") {
		af = abs(af)
		if af == "" || excluded[af] {
			continue
		}
		actionfiles[af] = true
	}

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
	dockerContents, err := updateDockerfiles(dockerfiles, goVers)
	if err != nil {
		log.Fatalf("latest_go_ensurer: %s", err)
	}

	travisContents, err := updateTravisFiles(travisfiles, goVers)
	if err != nil {
		log.Fatalf("latest_go_ensurer: %s", err)
	}

	actionContents, err := updateGitHubActionFiles(actionfiles, goVers)
	if err != nil {
		log.Fatalf("latest_go_ensurer: %s", err)
	}

	var contents []fileContent
	contents = append(contents, dockerContents...)
	contents = append(contents, travisContents...)
	contents = append(contents, actionContents...)

	sort.Slice(contents, func(i, j int) bool {
		return contents[i].origFP < contents[j].origFP
	})
	for _, fc := range contents {
		err := ioutil.WriteFile(fc.origFP, fc.contentsToWrite, 0644)
		if err != nil {
			log.Fatalf("latest_go_ensurer: unable to write new updated contents to %#v: %s", fc.origFP, err)
		}
	}
}

type fileContent struct {
	origFP          string
	contentsToWrite []byte
}

func getLatestGoVersion() (string, error) {
	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("https://golang.org/dl/?mode=json")
	if err != nil {
		return "", fmt.Errorf("unable to get list of Go releases from the golang.org/dl API: %s", err)
	}
	b, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return "", fmt.Errorf("unable to read body golang.org/dl API response: %s", err)
	}
	var releases []goRelease
	err = json.Unmarshal(b, &releases)
	if err != nil {
		return "", fmt.Errorf("unable to JSON parse golang.org/dl API response: %s", err)
	}
	for _, rel := range releases {
		if rel.Stable {
			return rel.Version[len("go"):], nil
		}
	}
	return "", fmt.Errorf("no stable release found in golang.org/dl API response")
}

type goRelease struct {
	Version string `json:"version"`
	Stable  bool   `json:"stable"`
}

func abs(fp string) string {
	out, err := filepath.Abs(filepath.Clean(fp))
	if err != nil {
		log.Fatalf("latest_go_ensurer: unable to get absolute path of %#v: %s", fp, err)
	}
	return out
}
