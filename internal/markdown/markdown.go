package markdown

import (
	"html/template"
	"net/url"
	"strings"

	_ "github.com/alecthomas/chroma/v2"
	"github.com/valyala/bytebufferpool"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
)

func Images(s string) ([]*url.URL, error) {
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

func Text(s string) (string, error) {
	b := bytebufferpool.Get()
	defer bytebufferpool.Put(b)

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
	if err := ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		switch n := n.(type) {
		case *ast.Text:
			if entering {
				_, _ = b.Write(n.Segment.Value([]byte(s)))
			}
		case *ast.String:
			if entering {
				_, _ = b.Write(n.Value)
			}

		case *ast.Paragraph:
			if !entering {
				_ = b.WriteByte(' ')
			}
		}
		return ast.WalkContinue, nil
	}); err != nil {
		return "", err
	}
	return strings.TrimSpace(b.String()), nil
}

func HTML(s string) (template.HTML, error) {
	b := bytebufferpool.Get()
	defer bytebufferpool.Put(b)

	md := goldmark.New(goldmark.WithExtensions(
		extension.GFM,
		highlighting.NewHighlighting(highlighting.WithStyle("monokai")),
		extension.NewTypographer()),
	)
	if err := md.Convert([]byte(s), b); err != nil {
		return "", err
	}
	return template.HTML(b.String()), nil
}
