package model

import (
	"html/template"
	"net/url"
	"strings"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
)

type Note struct {
	ID        string    `json:"id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

func (n *Note) HTML() template.HTML {
	buf := new(strings.Builder)
	md := goldmark.New(goldmark.WithExtensions(extension.GFM, extension.NewTypographer()))
	if err := md.Convert([]byte(n.Body), buf); err != nil {
		panic(err)
	}
	return template.HTML(buf.String())
}

func (n *Note) Description() string {
	source := []byte(n.Body)
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
	node := md.Parser().Parse(text.NewReader(source))
	buf := new(strings.Builder)
	if err := ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		switch n := n.(type) {
		case *ast.Text:
			if entering {
				buf.Write(n.Segment.Value(source))
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
		panic(err)
	}
	return strings.TrimSpace(buf.String())
}

func (n *Note) Images() []*url.URL {
	source := []byte(n.Body)
	node := goldmark.New(goldmark.WithExtensions(extension.GFM, extension.NewTypographer())).Parser().Parse(text.NewReader(source))
	var images []*url.URL
	if err := ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if n, ok := n.(*ast.Image); ok && entering {
			u, err := url.Parse(string(n.Destination))
			if err == nil {
				images = append(images, u)
			}
		}
		return ast.WalkContinue, nil
	}); err != nil {
		panic(err)
	}
	return images
}
