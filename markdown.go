package main

import (
	"html/template"
	"net/url"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
)

func markdownImages(s string) ([]*url.URL, error) {
	var images []*url.URL
	node := goldmark.New(goldmark.WithExtensions(extension.GFM, extension.NewTypographer())).Parser().Parse(text.NewReader([]byte(s)))
	if err := ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if n, ok := n.(*ast.Image); ok && entering {
			u, err := url.Parse(string(n.Destination))
			if err == nil {
				images = append(images, u)
			}
		}
		return ast.WalkContinue, nil
	}); err != nil {
		return nil, err
	}
	return images, nil
}

func markdownText(s string) (string, error) {
	md := goldmark.New(goldmark.WithExtensions(
		extension.GFM,
		extension.NewTypographer(
			extension.WithTypographicSubstitutions(
				extension.TypographicSubstitutions{
					extension.LeftSingleQuote:  []byte(`‘`),
					extension.RightSingleQuote: []byte(`’`),
					extension.LeftDoubleQuote:  []byte(`“`),
					extension.RightDoubleQuote: []byte(`”`),
					extension.EnDash:           []byte(`–`),
					extension.EmDash:           []byte(`—`),
					extension.Ellipsis:         []byte(`…`),
					extension.LeftAngleQuote:   []byte(`«`),
					extension.RightAngleQuote:  []byte(`»`),
					extension.Apostrophe:       []byte(`’`),
				}))))
	node := md.Parser().Parse(text.NewReader([]byte(s)))
	buf := new(strings.Builder)
	if err := ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		switch n := n.(type) {
		case *ast.Text:
			if entering {
				buf.Write(n.Segment.Value([]byte(s)))
			}
		case *ast.String:
			if entering {
				buf.Write(n.Value)
			}

		case *ast.Paragraph:
			if !entering {
				buf.WriteByte(' ')
			}
		}
		return ast.WalkContinue, nil
	}); err != nil {
		return "", err
	}
	return strings.TrimSpace(buf.String()), nil
}

func markdownHTML(s string) (template.HTML, error) {
	buf := new(strings.Builder)
	md := goldmark.New(goldmark.WithExtensions(extension.GFM, extension.NewTypographer()))
	if err := md.Convert([]byte(s), buf); err != nil {
		return "", err
	}
	return template.HTML(buf.String()), nil
}
