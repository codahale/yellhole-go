package model

import (
	"fmt"
	"html/template"
	"net/url"
	"testing"
)

func TestNoteHTML(t *testing.T) {
	note := Note{Body: "It's ~~not~~ _electric_!"}
	if expected, actual := template.HTML("<p>It&rsquo;s <del>not</del> <em>electric</em>!</p>\n"), note.HTML(); expected != actual {
		t.Errorf("expected %q, but was %q", expected, actual)
	}
}

func TestNoteDescription(t *testing.T) {
	note := Note{Body: "It's _electric_!\n\nBoogie woogie woogie."}
	if expected, actual := "It’s electric! Boogie woogie woogie.", note.Description(); expected != actual {
		t.Errorf("expected %q, but was %q", expected, actual)
	}
}

func TestNoteImages(t *testing.T) {
	a, _ := url.Parse("/doink.png")
	b, _ := url.Parse("http://example.com/cool.bmp")

	note := Note{Body: fmt.Sprintf("Hello!\n\n![](%s)\n\n![](%s)", a, b)}
	actual := note.Images()

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
