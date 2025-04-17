package main

import (
	"fmt"
	"html/template"
	"net/url"
	"testing"
)

func TestMarkdownHTML(t *testing.T) {
	html, err := markdownHTML("It's ~~not~~ _electric_!")
	if err != nil {
		t.Fatal(err)
	}

	if got, want := html, template.HTML("<p>It&rsquo;s <del>not</del> <em>electric</em>!</p>\n"); got != want {
		t.Errorf("markdownHTML(s) = %q, want = %q", got, want)
	}
}

func TestMarkdownText(t *testing.T) {
	text, err := markdownText("It's _electric_!\n\nBoogie woogie woogie.")
	if err != nil {
		t.Fatal(err)
	}

	if got, want := text, "It’s electric! Boogie woogie woogie."; got != want {
		t.Errorf("markdownText(s) = %q, want = %q", got, want)
	}
}

func TestMarkdownImages(t *testing.T) {
	a, _ := url.Parse("/doink.png")
	b, _ := url.Parse("http://example.com/cool.bmp")

	images, err := markdownImages(fmt.Sprintf("Hello!\n\n![](%s)\n\n![](%s)", a, b))
	if err != nil {
		t.Fatal(err)
	}

	if got, want := 2, len(images); got != want {
		t.Fatalf("len(images) = %d, want = %d", got, want)
	}

	if got, want := images[0].String(), a.String(); got != want {
		t.Errorf("images[0].String() = %q, want = %q", got, want)
	}

	if got, want := images[1].String(), b.String(); got != want {
		t.Errorf("images[1].String() = %q, want = %q", got, want)
	}
}
