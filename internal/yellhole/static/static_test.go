package static

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestStatic(t *testing.T) {
	mux := http.NewServeMux()
	Register(mux)

	ts := httptest.NewServer(mux)
	defer ts.Close()

	u, _ := url.JoinPath(ts.URL, "favicon.ico")
	resp, err := http.Get(u)
	if err != nil {
		t.Fatal(err)
	}

	if want, got := "image/x-icon", resp.Header.Get("content-type"); want != got {
		t.Errorf("wanted Content-Type: %s but was %q", want, got)
	}
}
