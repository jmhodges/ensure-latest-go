package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var (
	excludeFiles     = flag.String("excludeFiles", "", "List of comma-seperated file paths to not update to the latest Go")
	dockerfilesInput = flag.String("dockerfiles", "", "List of comma-separate file paths to Dockerfiles")
	travisfilesInput = flag.String("travisfiles", "", "List of comma-separate file paths to .travis.yml Travis config files")
	actionfilesInput = flag.String("actionfiles", "", "List of comma-separate file paths to GitHub Action config files")
)

func main() {
	flag.Parse()
	log.Println("FIXME PWD is", os.Getenv("PWD"))

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
	travisfiles := make(map[string]bool)
	for _, tf := range strings.Split(*travisfilesInput, ",") {
		tf = abs(tf)
		if tf == "" || excluded[tf] {
			continue
		}
		travisfiles[tf] = true
	}

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
	// FIXME implement
	return "1.22", nil
}

func abs(fp string) string {
	out, err := filepath.Abs(filepath.Clean(fp))
	if err != nil {
		log.Fatalf("latest_go_ensurer: unable to get absolute path of %#v: %s", fp, err)
	}
	return out
}
