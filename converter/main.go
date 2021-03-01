package main

import (
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"log"
	"os"
)

type CFRDoc struct {
	XMLName xml.Name `xml:"CFRDOC"`
	Title   Title    `xml:"TITLE"`
}

type Title struct {
	Chapters []Chapter `xml:"CHAPTER"`
}

type Chapter struct {
	Subchaps []Subchapter `xml:"SUBCHAP"`
}

type Subchapter struct {
	Parts  []Part `xml:"PART"`
	Header string `xml:"HD"`
}

type Part struct {
	Sections []Section `xml:"SECTION"`
	Subparts []Subpart `xml:"SUBPART"`
	Header   string    `xml:"HD"`
}

type Subpart struct {
	Header   string    `xml:"HD"`
	Children *Children `xml:",any"`
}

type Children struct {
	Kids []interface{}
}

func (c *Children) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	switch start.Name.Local {
	case "SUBJGRP":
		child := SubjectGroup{}
		if err := d.DecodeElement(&child, &start); err != nil {
			return err
		}
		c.Kids = append(c.Kids, child)
	case "SECTION":
		child := Section{}
		if err := d.DecodeElement(&child, &start); err != nil {
			return err
		}
		c.Kids = append(c.Kids, child)
	default:
		child := Node{}
		if err := d.DecodeElement(&child, &start); err != nil {
			return err
		}
		c.Kids = append(c.Kids, child)
	}

	return nil
}

type SubjectGroup struct {
	XMLName  xml.Name
	Sections []Section `xml:"SECTION"`
	Header   string    `xml:"HD"`
}

type Section struct {
	XMLName    xml.Name
	Number     string      `xml:"SECTNO"`
	Subject    string      `xml:"SUBJECT"`
	Paragraphs []Paragraph `xml:",any" json:"children"`
}

type Paragraph struct {
	XMLName xml.Name
	Content string `xml:",innerxml" json:"text"`
}

type Node struct {
	Name     xml.StartElement
	Children []*Node
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

	json, err := json.MarshalIndent(doc, "", "    ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(json))
}
