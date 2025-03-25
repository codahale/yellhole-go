// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Adapted from encoding/xml/read_test.go.

package main

import (
	"encoding/xml"
	"time"
)

const (
	atomContentType = "application/atom+xml"
)

type atomFeed struct {
	XMLName  xml.Name     `xml:"http://www.w3.org/2005/Atom feed"`
	Title    string       `xml:"title"`
	Subtitle string       `xml:"subtitle"`
	ID       string       `xml:"id"`
	Link     []atomLink   `xml:"link"`
	Updated  atomTimeStr  `xml:"updated"`
	Author   *atomPerson  `xml:"author"`
	Entry    []*atomEntry `xml:"entry"`
}

type atomEntry struct {
	Title     string      `xml:"title"`
	ID        string      `xml:"id"`
	Link      []atomLink  `xml:"link"`
	Published atomTimeStr `xml:"published"`
	Updated   atomTimeStr `xml:"updated"`
	Author    *atomPerson `xml:"author"`
	Summary   *atomText   `xml:"summary"`
	Content   *atomText   `xml:"content"`
}

type atomLink struct {
	Rel      string `xml:"rel,attr,omitempty"`
	Href     string `xml:"href,attr"`
	Type     string `xml:"type,attr,omitempty"`
	HrefLang string `xml:"hreflang,attr,omitempty"`
	Title    string `xml:"title,attr,omitempty"`
	Length   uint   `xml:"length,attr,omitempty"`
}

type atomPerson struct {
	Name     string `xml:"name"`
	URI      string `xml:"uri,omitempty"`
	Email    string `xml:"email,omitempty"`
	InnerXML string `xml:",innerxml"`
}

type atomText struct {
	Type string `xml:"type,attr"`
	Body string `xml:",chardata"`
}

type atomTimeStr string

func atomTime(t time.Time) atomTimeStr {
	return atomTimeStr(t.Format("2006-01-02T15:04:05-07:00"))
}
