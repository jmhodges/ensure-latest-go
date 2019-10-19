package main

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDockerfileFromUpdate(t *testing.T) {
	testcases := []struct {
		origLine    string
		newImageTag string
		expected    string
	}{
		{
			"from golangadf aa",
			"1.13",
			"from golangadf aa",
		},
		{
			"from    golang:1.1.1 aa",
			"1.13.1",
			"from    golang:1.13.1 aa",
		},
		{
			"from golang",
			"1.2",
			"from golang:1.2",
		},
		{
			"from golang # foobar",
			"1.2",
			"from golang:1.2 # foobar",
		},
		{
			"from golang# foobar",
			"1.2",
			"from golang:1.2# foobar",
		},
		{
			"FROM    golang# foobar",
			"1.2",
			"FROM    golang:1.2# foobar",
		},
		{
			"FROM golang:1.13.1",
			"1.1",
			"FROM golang:1.1",
		},
		{
			"from golang:1.13.1-alpine",
			"1.13.3",
			"from golang:1.13.3-alpine",
		},
		{
			"from golang:1.-werd",
			"1.13.3",
			"from golang:1.13.3",
		},
	}

	for i, tc := range testcases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			actual, err := updateDockerfileFromLine([]byte(tc.origLine), tc.newImageTag)
			if err != nil {
				t.Errorf("updateDockerfileFromLine error: %s", err)
				return
			}
			if tc.expected != string(actual) {
				t.Errorf("want %#v, got %#v", tc.expected, string(actual))
			}
		})
	}
}

func TestTravisGoldenPath(t *testing.T) {
	testcases := []struct {
		input    string
		expected string
	}{
		{
			input: `language: go
go:
  - 1.13.1

sudo: required

services:
  - docker

branches:
  only:
    - master
    - /^test_/
    - /^test-/

install:
  - go test -race -i .

script:
  - go test -race . && GOOS=linux GOARCH=amd64 go build -ldflags "-X main.buildSHA=${TRAVIS_COMMIT}" . && ./travis_docker_push.sh
`,
			expected: `language: go
go:
- "1.22"
sudo: required
services:
- docker
branches:
  only:
  - master
  - /^test_/
  - /^test-/
install:
- go test -race -i .
script:
- go test -race . && GOOS=linux GOARCH=amd64 go build -ldflags "-X main.buildSHA=${TRAVIS_COMMIT}" . && ./travis_docker_push.sh
`,
		},
		{
			input: `language: go
go:
  - 1.13.1
  - 1.10.0

foobar: foo
`,
			expected: `language: go
go:
- 1.13.1
- 1.10.0
- "1.22"
foobar: foo
`,
		},
		{
			input: `language: go
go: 1.13.1
branches:
- nope
`,
			expected: `language: go
go: "1.22"
branches:
- nope
`,
		},

		{
			input: `language: go
branches:
- nope
`,
			expected: `language: go
branches:
- nope
`,
		},
	}

	for i, tc := range testcases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			actualBytes, err := updateSingleTravisFile("fake.yml", []byte(tc.input), "1.22")
			if err != nil {
				t.Fatalf("updateSingleTravisFile: %s", err)
			}
			actual := string(actualBytes)
			if tc.expected != actual {
				t.Errorf("github action file update failed: %s (%s)", cmp.Diff(tc.expected, actual), actual)
			}
		})
	}

}
