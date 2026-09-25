package subsonic

import (
	"strings"
	"testing"
)

func TestListParses(t *testing.T) {
	body := `{"subsonic-response":{"status":"ok","internetRadioStations":{"internetRadioStation":[` +
		`{"id":"1","name":"A","streamUrl":"http://a","homePageUrl":"http://h"}]}}}`
	c := &Client{User: "admin", Call: func(uri string) (string, error) { return body, nil }}
	radios, err := c.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(radios) != 1 || radios[0].Name != "A" || radios[0].HomepageURL != "http://h" {
		t.Fatalf("%+v", radios)
	}
}

func TestListError(t *testing.T) {
	body := `{"subsonic-response":{"status":"failed","error":{"code":40,"message":"wrong username"}}}`
	c := &Client{User: "admin", Call: func(uri string) (string, error) { return body, nil }}
	if _, err := c.List(); err == nil {
		t.Fatal("expected error")
	}
}

func TestCreateBuildsURI(t *testing.T) {
	var got string
	c := &Client{User: "admin", Call: func(uri string) (string, error) {
		got = uri
		return `{"subsonic-response":{"status":"ok"}}`, nil
	}}
	if err := c.Create("My Station", "http://s", "http://h"); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, "createInternetRadioStation?") {
		t.Fatalf("uri=%q", got)
	}
	for _, want := range []string{"name=My+Station", "streamUrl=http%3A%2F%2Fs", "homepageUrl=http%3A%2F%2Fh", "u=admin"} {
		if !strings.Contains(got, want) {
			t.Fatalf("uri=%q missing %q", got, want)
		}
	}
}
