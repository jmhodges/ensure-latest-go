package main

import (
	"bytes"
	"fmt"

	"gopkg.in/jmhodges/yaml.v2"
)

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

func yamlMarshal(obj yaml.MapSlice) ([]byte, error) {
	buf := &bytes.Buffer{}
	enc := yaml.NewEncoder(buf)
	enc.SetLineLength(-1) // Disable line wrap
	err := enc.Encode(obj)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
