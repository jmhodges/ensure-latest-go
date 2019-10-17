package main

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDockerfileFromUpdate(t *testing.T) {
	testcases := []struct {
		origLine          string
		imageNameToUpdate string
		newImageTag       string
		expected          string
	}{
		{
			"from golangadf aa",
			"golang",
			"1.13",
			"from golangadf aa",
		},
		{
			"from    golang2:1.1 aa",
			"golang2",
			"1.13",
			"from    golang2:1.13 aa",
		},
		{
			"from golang2",
			"golang2",
			"1.2",
			"from golang2:1.2",
		},
		{
			"from golang2 # foobar",
			"golang2",
			"1.2",
			"from golang2:1.2 # foobar",
		},
		{
			"from golang2# foobar",
			"golang2",
			"1.2",
			"from golang2:1.2# foobar",
		},
		{
			"FROM    golang2# foobar",
			"golang2",
			"1.2",
			"FROM    golang2:1.2# foobar",
		},
		{
			"FROM golang:1.13.1",
			"golang",
			"1.1",
			"FROM golang:1.1",
		},
	}

	for i, tc := range testcases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			actual, err := updateDockerfileFromLine([]byte(tc.origLine), tc.imageNameToUpdate, tc.newImageTag)
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

func TestGitHubActionGoldenPath(t *testing.T) {
	testcases := []struct {
		input    string
		expected string
	}{
		{
			input: `name: Go
on: 
  push:
    branches: 
      - foobar
  pull_request:
    branches: 
      - master

jobs:
  test:
    name: Run Go build
    runs-on: ubuntu-latest
    steps:

    - name: Set up Go
      uses: actions/setup-go@v1
      with:
        go-version: 1.13.1
      id: go

    - name: Check out code into the Go module directory
      uses: actions/checkout@v1

    - name: Build
      run: go install -race ./...
`,
			expected: `name: Go
"on":
  push:
    branches:
    - foobar
  pull_request:
    branches:
    - master
jobs:
  test:
    name: Run Go build
    runs-on: ubuntu-latest
    steps:
    - name: Set up Go
      uses: actions/setup-go@v1
      with:
        go-version: "1.22"
      id: go
    - name: Check out code into the Go module directory
      uses: actions/checkout@v1
    - name: Build
      run: go install -race ./...
`,
		},
		{
			input: `name: Go
on: 
  push:
    branches: 
      - foobar
  pull_request:
    branches: 
      - master

jobs:
  test:
    name: Run Go build
    runs-on: ubuntu-latest
    strategy:
      matrix:
        go: [ '1.9', '1.10.x' ]
    steps:
    - name: Set up Go
      uses: actions/setup-go@v1
      with:
        go-version: ${{ matrix.go }}
      id: go

    - name: Check out code into the Go module directory
      uses: actions/checkout@v1

    - name: Build
      run: go install -race ./...
`,
			expected: `name: Go
"on":
  push:
    branches:
    - foobar
  pull_request:
    branches:
    - master
jobs:
  test:
    name: Run Go build
    runs-on: ubuntu-latest
    strategy:
      matrix:
        go:
        - "1.9"
        - 1.10.x
        - "1.22"
    steps:
    - name: Set up Go
      uses: actions/setup-go@v1
      with:
        go-version: ${{ matrix.go }}
      id: go
    - name: Check out code into the Go module directory
      uses: actions/checkout@v1
    - name: Build
      run: go install -race ./...
`,
		},
	}

	for i, tc := range testcases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			actualBytes, err := updateSingleGitHubActionFile("fake.yml", []byte(tc.input), "1.22")
			if err != nil {
				t.Fatalf("updateSingleGitHubActionFile: %s", err)
			}
			actual := string(actualBytes)
			if tc.expected != actual {
				t.Errorf("github action file update failed: %s (%s)", cmp.Diff(tc.expected, actual), actual)
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
- 1.13.1
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
  - 1.22
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
go: 1.22
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
