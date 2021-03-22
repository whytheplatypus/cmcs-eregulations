package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"testing"
)

// JSONEqual compares the JSON from two Readers.
// source: https://stackoverflow.com/questions/32408890/how-to-compare-two-json-requests
func JSONEqual(compare, to io.Reader) error {
	var have, expected map[string]interface{}
	d := json.NewDecoder(compare)
	if err := d.Decode(&have); err != nil {
		return err
	}
	d = json.NewDecoder(to)
	if err := d.Decode(&expected); err != nil {
		return err
	}
	return compareJSON(have, expected)
}

func compareJSON(compare, expected interface{}) error {
	switch expected.(type) {
	//bool, for JSON booleans
	//float64, for JSON numbers
	//string, for JSON strings
	case []interface{}:
		have, ok := compare.([]interface{})
		if !ok {
			return fmt.Errorf("Types don't match %#v", compare)
		}
		want := expected.([]interface{})
		if len(have) < len(want) {
			return fmt.Errorf("Missing some elements of the array %d %d", len(have), len(want))
		}
		for i, val := range want {
			var notfound error
			for _, v := range have {
				if notfound = compareJSON(v, val); notfound == nil {
					break
				}
			}
			if notfound != nil {
				return fmt.Errorf("Missing node %d %s", i, notfound)
			}
		}
	case map[string]interface{}:
		have, ok := compare.(map[string]interface{})
		if !ok {
			return fmt.Errorf("Types don't match")
		}
		for key, val := range expected.(map[string]interface{}) {
			v, ok := have[key]
			if !ok {
				return fmt.Errorf("Missing key: %s", key)
			}
			if err := compareJSON(v, val); err != nil {
				return fmt.Errorf("Failed for key %s: %s", key, err)
			}

		}
	}
	return nil
}

func TestCFRDecode(t *testing.T) {
	f, err := os.Open("CFR-2019-title42-vol4.xml")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	d := xml.NewDecoder(f)
	doc := &CFRDoc{}
	if err := d.Decode(doc); err != nil {
		t.Error(err)
	}
	json, err := json.MarshalIndent(doc.Part("433"), "", "    ")
	if err != nil {
		t.Fatal(err)
	}

	expected, err := os.ReadFile("output.json/regulation/433/2019-annual-433")
	if err != nil {
		t.Fatal(err)
	}

	if err := JSONEqual(bytes.NewBuffer(json), bytes.NewBuffer(expected)); err != nil {
		t.Error("results are not equal", err)
	}
}
