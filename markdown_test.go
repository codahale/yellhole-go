package main

import (
	"fmt"
	"html/template"
	"net/url"
	"testing"
)

func TestHTML(t *testing.T) {
	actual, err := markdownHTML("It's ~~not~~ _electric_!")
	if err != nil {
		t.Fatal(err)
	}
	if expected := template.HTML("<p>It&rsquo;s <del>not</del> <em>electric</em>!</p>\n"); expected != actual {
		t.Errorf("expected %q, but was %q", expected, actual)
	}
}

func TestNoteDescription(t *testing.T) {
	actual, err := markdownText("It's _electric_!\n\nBoogie woogie woogie.")
	if err != nil {
		t.Fatal(err)
	}
	if expected := "It’s electric! Boogie woogie woogie."; expected != actual {
		t.Errorf("expected %q, but was %q", expected, actual)
	}
}

func TestNoteImages(t *testing.T) {
	a, _ := url.Parse("/doink.png")
	b, _ := url.Parse("http://example.com/cool.bmp")

	actual, err := markdownImages(fmt.Sprintf("Hello!\n\n![](%s)\n\n![](%s)", a, b))
	if err != nil {
		t.Fatal(err)
	}

	if len(actual) != 2 {
		t.Fatalf("expected 2 images but was %d", len(actual))
	}

	if expected, actual := a.String(), actual[0].String(); expected != actual {
		t.Errorf("expected %s but was %s", expected, actual)
	}

	if expected, actual := b.String(), actual[1].String(); expected != actual {
		t.Errorf("expected %s but was %s", expected, actual)
	}
}
