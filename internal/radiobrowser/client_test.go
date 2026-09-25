package radiobrowser

import (
	"errors"
	"testing"
)

type fakeDoer struct {
	calls   int
	fail    int
	lastURL string
}

func (f *fakeDoer) Do(method, url string, headers map[string]string, timeoutMs int32) (int, []byte, error) {
	f.calls++
	f.lastURL = url
	if f.calls <= f.fail {
		return 0, nil, errors.New("boom")
	}
	return 200, []byte(`[{"name":"A","url_resolved":"http://x"}]`), nil
}

func TestPageParses(t *testing.T) {
	d := &fakeDoer{}
	c := &Client{Doer: d, PageSize: 10}
	stations, err := c.Page(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(stations) != 1 || stations[0].Field("name") != "A" {
		t.Fatalf("bad %+v", stations)
	}
}

func TestPageRetriesMirrors(t *testing.T) {
	d := &fakeDoer{fail: 1}
	c := &Client{Doer: d, PageSize: 10, Mirrors: []string{"https://m1", "https://m2"}}
	if _, err := c.Page(0); err != nil {
		t.Fatal(err)
	}
	if d.calls != 2 {
		t.Fatalf("calls=%d", d.calls)
	}
}

func TestPageBuildsQuery(t *testing.T) {
	d := &fakeDoer{}
	c := &Client{Doer: d, PageSize: 10, HideBroken: true, BaseURL: "https://m1"}
	if _, err := c.Page(20); err != nil {
		t.Fatal(err)
	}
	want := "https://m1/json/stations/search?hidebroken=true&limit=10&offset=20"
	if d.lastURL != want {
		t.Fatalf("url=%q want %q", d.lastURL, want)
	}
}
