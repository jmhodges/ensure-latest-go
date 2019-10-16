package main

import (
	"fmt"
	"testing"
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
