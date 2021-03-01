package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
	"os"
	"reflect"
	"testing"
)

// JSONEqual compares the JSON from two Readers.
// source: https://stackoverflow.com/questions/32408890/how-to-compare-two-json-requests
func JSONEqual(a, b io.Reader) (bool, error) {
	var j, j2 interface{}
	d := json.NewDecoder(a)
	if err := d.Decode(&j); err != nil {
		return false, err
	}
	d = json.NewDecoder(b)
	if err := d.Decode(&j2); err != nil {
		return false, err
	}
	return reflect.DeepEqual(j2, j), nil
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
	json, err := json.MarshalIndent(doc, "", "    ")
	if err != nil {
		t.Fatal(err)
	}

	expected, err := os.ReadFile("output.json/regulation/433/2019-annual-433")
	if err != nil {
		t.Fatal(err)
	}

	if ok, err := JSONEqual(bytes.NewBuffer(json), bytes.NewBuffer(expected)); !ok || err != nil {
		t.Error("results are not equal")
	}
}
