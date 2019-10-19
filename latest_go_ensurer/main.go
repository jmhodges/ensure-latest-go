package main

import (
	"bytes"
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
	excludeFiles := os.Getenv("INPUT_EXCLUDES")
	excluded := make(map[string]bool)
	for _, ef := range strings.Split(excludeFiles, ",") {
		ef := abs(ef)
		excluded[ef] = true
	}

	dockerfiles := gatherDockerfiles(excluded)
	travisfiles := gatherTravisfiles(excluded)
	actionVersions, err := gatherGitHubActionGoVersion(ghActionVersionFile, excluded)

	if err != nil {
		log.Fatalf("latest_go_ensurer: unable to parse .github/versions/go: %s", err)
	}

	if len(dockerfiles)+len(travisfiles) == 0 {
		log.Fatalf("latest_go_ensurer: no files given to update. Set the dockerfiles, or travisfiles arguments in your GitHub Action workflow")
	}

	goVers, err := getLatestGoVersion()
	if err != nil {
		log.Fatalf("latest_go_ensurer: %s", err)
	}

	fmt.Println(goVers) // for set-output in the GitHub Action

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

	actionContents, err := updateGitHubActionVersionFile(ghActionVersionFile, actionVersions, goVers)
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
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("golang.org/dl API returned HTTP status code %d instead of a 200", resp.StatusCode)
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

func gatherDockerfiles(excluded map[string]bool) map[string]bool {
	dockerfilesInput := strings.TrimSpace(os.Getenv("INPUT_DOCKERFILES"))
	var dockerpaths []string
	if len(dockerfilesInput) != 0 {
		dockerpaths = strings.Split(dockerfilesInput, ",")
	} else {
		filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
			if info.Name() == "Dockerfile" {
				dockerpaths = append(dockerpaths, path)
			}
			return nil
		})
	}
	return uniqUnexcludedPaths(dockerpaths, excluded)
}

func gatherTravisfiles(excluded map[string]bool) map[string]bool {
	travisfilesInput := strings.TrimSpace(os.Getenv("INPUT_TRAVISFILES"))
	var travispaths []string
	if len(travisfilesInput) != 0 {
		travispaths = strings.Split(travisfilesInput, ",")
	} else {
		fp := ".travis.yml"
		_, err := os.Stat(fp)
		if err == nil {
			travispaths = append(travispaths, fp)
		}
	}
	return uniqUnexcludedPaths(travispaths, excluded)
}

const ghActionVersionFile = ".github/versions/go"

func gatherGitHubActionGoVersion(excluded map[string]bool) (string, error) {
	if excluded[ghActionVersionFile] {
		return "", nil
	}
	b, err := ioutil.ReadFile(ghActionVersionFile)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", nil
	}
	return string(bytes.TrimSpace(b)), nil
}

func uniqUnexcludedPaths(paths []string, excluded map[string]bool) map[string]bool {
	files := make(map[string]bool)
	for _, fp := range paths {
		fp = abs(fp)
		if fp == "" || excluded[fp] {
			continue
		}
		files[fp] = true
	}
	return files
}
