package main

import (
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
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
	Label    []string  `json:"label"`
}

type Children struct {
	Parent interface{}
	Kids   []interface{}
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
	case "P":
		child := &Paragraph{}
		if err := d.DecodeElement(child, &start); err != nil {
			return err
		}
		child.Parent = c.Parent
		child.Siblings = c.Kids
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
	XMLName  xml.Name   `json:"meta"`
	Sections []*Section `xml:"SECTION" json:"children"`
	Header   string     `xml:"HD"`
}

type Section struct {
	XMLName  xml.Name  `json:"meta"`
	Number   string    `xml:"SECTNO"`
	Subject  string    `xml:"SUBJECT"`
	Children *Children `xml:",any" json:"children"`
}

func (s *Section) Label() []string {
	ref := strings.TrimSpace(strings.TrimPrefix(s.Number, "§"))
	return strings.Split(ref, ".")
}

func (s *Section) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	type section2 Section
	p := (*section2)(s)
	p.Children = &Children{}
	p.Children.Parent = s
	if err := d.DecodeElement(p, &start); err != nil {
		return err
	}
	return nil
}

type Paragraph struct {
	XMLName  xml.Name      `json:"meta"`
	Content  string        `xml:",innerxml" json:"text"`
	Parent   interface{}   `json:"-"`
	Siblings []interface{} `json:"-"`
}

// a, 1, roman, upper, italic int, italic roman
var alpha = regexp.MustCompile(`\(([a-z]+)\)`)
var num = regexp.MustCompile(`\((\d+)\)`)
var upper = regexp.MustCompile(`\(([A-Z]+)\)`)
var roman = regexp.MustCompile(`\((ix|iv|v?i{0,3})\)`)

func (p *Paragraph) Label() []string {
	re := regexp.MustCompile(`^([\(\w+\)]+)`)
	pLabel := re.FindStringSubmatch(p.Content)
	if len(pLabel) < 2 {
		return nil
	}
	parent, ok := p.Parent.(Labeled)
	if !ok {
		return pLabel
	}
	l := parent.Label()
	return append(l, pLabel...)
}

func (p *Paragraph) MarshalJSON() ([]byte, error) {
	type p2 struct {
		XMLName xml.Name `json:"meta"`
		Content string   `xml:",innerxml" json:"text"`
		Label   []string `json:"label"`
	}
	toMarshal := &p2{
		p.XMLName,
		p.Content,
		p.Label(),
	}
	b, err := json.Marshal(toMarshal)
	if err != nil {
		return b, err
	}
	return b, nil
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
	part := doc.Part("433")
	json, err := json.MarshalIndent(part, "", "    ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(json))
}

type Labeled interface {
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
