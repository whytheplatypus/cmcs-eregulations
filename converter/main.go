package main

import (
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

type CFRDoc struct {
	XMLName xml.Name `xml:"CFRDOC" json:"meta"`
	Title   *Title   `xml:"TITLE"`
}

type Title struct {
	Chapters []*Chapter `xml:"CHAPTER"`
}

type Chapter struct {
	Subchaps []*Subchapter `xml:"SUBCHAP"`
}

type Subchapter struct {
	Parts  []*Part `xml:"PART"`
	Header string  `xml:"HD"`
}

type NodeType string

func (nt *NodeType) MarshalText() (text []byte, err error) {
	return []byte("reg_text"), nil
}

type Part struct {
	//Sections []Section `xml:"SECTION"`
	//Subparts []Subpart `xml:"SUBPART"`
	Children *Children `xml:",any" json:"children"`
	Header   string    `xml:"HD" json:"title"`
	Text     string    `xml:",chardata" json:"text"`
	NodeType NodeType  `json:"node_type"`
	Label    []string  `json:"label"`
}

type Subpart struct {
	Header   string    `xml:"HD"`
	Children *Children `xml:",any" json:"children"`
	Label    []string
}

type Children struct {
	Kids []interface{}
}

func (c *Children) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	switch start.Name.Local {
	case "SUBPART":
		child := &Subpart{}
		if err := d.DecodeElement(child, &start); err != nil {
			return err
		}
		c.Kids = append(c.Kids, child)
	case "SUBJGRP":
		child := &SubjectGroup{}
		if err := d.DecodeElement(child, &start); err != nil {
			return err
		}
		c.Kids = append(c.Kids, child)
	case "SECTION":
		child := &Section{}
		if err := d.DecodeElement(child, &start); err != nil {
			return err
		}
		c.Kids = append(c.Kids, child)
	default:
		child := &Node{}
		if err := d.DecodeElement(child, &start); err != nil {
			return err
		}
		c.Kids = append(c.Kids, child)
	}

	return nil
}

func (c *Children) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.Kids)
}

type SubjectGroup struct {
	XMLName  xml.Name  `json:"meta"`
	Sections []Section `xml:"SECTION" json:"children"`
	Header   string    `xml:"HD"`
}

type Section struct {
	XMLName    xml.Name    `json:"meta"`
	Number     string      `xml:"SECTNO"`
	Subject    string      `xml:"SUBJECT"`
	Paragraphs []Paragraph `xml:",any" json:"children"`
}

type Paragraph struct {
	XMLName xml.Name `json:"meta"`
	Content string   `xml:",innerxml" json:"text"`
}

type Node struct {
	Name     xml.StartElement `json:"meta"`
	Children []*Node          `json:"children"`
	Content  []string
}

//Really this would be the most useful as post processing of Content from the more structured solution above.
func (c *Node) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	c.Name = start
	for t, _ := d.Token(); t != nil; t, _ = d.Token() {
		chars, ok := t.(xml.CharData)
		if ok {
			c.Content = append(c.Content, string(chars))
			continue
		}
		startEl, ok := t.(xml.StartElement)
		if ok {
			child := &Node{}
			if err := d.DecodeElement(child, &startEl); err != nil {
				return err
			}
			c.Children = append(c.Children, child)
		}
	}
	return nil
}

func main() {
	flag.Parse()
	filename := flag.Arg(0)
	f, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	d := xml.NewDecoder(f)

	doc := &CFRDoc{}
	//doc := &Node{}
	if err := d.Decode(doc); err != nil {
		log.Fatal(err)
	}
	doc.FillLabels()
	part := doc.Part("433")
	json, err := json.MarshalIndent(part, "", "    ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(json))
}

type Labeled interface {
	SetLabel([]string)
	Label() []string
}

func (doc *CFRDoc) FillLabels() {
	for _, chap := range doc.Title.Chapters {
		for _, subchap := range chap.Subchaps {
			for _, part := range subchap.Parts {
				part.Label = []string{part.Header}
				for _, child := range part.Children.Kids {
					subpart, ok := child.(*Subpart)
					if ok {
						subpart.Label = append(part.Label, subpart.Header)
					}
				}
			}
		}
	}
}

func (doc *CFRDoc) Part(num string) *Part {
	for _, chap := range doc.Title.Chapters {
		for _, subchap := range chap.Subchaps {
			for _, part := range subchap.Parts {
				if strings.Contains(part.Header, num) {
					return part
				}
			}
		}
	}
	return nil
}
