package filter

import (
	"testing"

	"github.com/ruuddeenen/navidrome-radio-sync-plugin/internal/radiobrowser"
	"github.com/ruuddeenen/navidrome-radio-sync-plugin/internal/settings"
)

func st(json string) radiobrowser.Station {
	s, err := radiobrowser.ParseStations([]byte("[" + json + "]"))
	if err != nil {
		panic(err)
	}
	return s[0]
}

func TestHideBroken(t *testing.T) {
	s := settings.Defaults()
	if d := Apply(s, st(`{"name":"A","url":"http://x","lastcheckok":0}`)); d.Keep {
		t.Fatal("broken should drop")
	}
	if d := Apply(s, st(`{"name":"A","url":"http://x","lastcheckok":1}`)); !d.Keep {
		t.Fatal("ok should keep")
	}
}

func TestRequireCountryCodeAndInclude(t *testing.T) {
	s := settings.Defaults()
	s.RequireCountryCode = true
	if d := Apply(s, st(`{"name":"A","url":"http://x","lastcheckok":1,"countrycode":""}`)); d.Keep {
		t.Fatal("no code should drop")
	}
	s.IncludeCountryCodes = []string{"NL", "DE"}
	if d := Apply(s, st(`{"name":"A","url":"http://x","lastcheckok":1,"countrycode":"US"}`)); d.Keep {
		t.Fatal("US should drop")
	}
	if d := Apply(s, st(`{"name":"A","url":"http://x","lastcheckok":1,"countrycode":"nl"}`)); !d.Keep {
		t.Fatal("NL should keep")
	}
}

func TestTagFilters(t *testing.T) {
	s := settings.Defaults()
	s.ExcludeTags = []string{"religious"}
	if d := Apply(s, st(`{"name":"A","url":"http://x","lastcheckok":1,"tags":"pop, religious"}`)); d.Keep {
		t.Fatal("excluded tag should drop")
	}
	s = settings.Defaults()
	s.IncludeTags = []string{"jazz"}
	if d := Apply(s, st(`{"name":"A","url":"http://x","lastcheckok":1,"tags":"pop"}`)); d.Keep {
		t.Fatal("missing include tag should drop")
	}
	if d := Apply(s, st(`{"name":"A","url":"http://x","lastcheckok":1,"tags":"Jazz, funk"}`)); !d.Keep {
		t.Fatal("jazz should keep")
	}
}

func TestBitrateAndCodec(t *testing.T) {
	s := settings.Defaults()
	s.MinBitrate = 128
	s.ExcludeCodecs = []string{"mp3"}
	if d := Apply(s, st(`{"name":"A","url":"http://x","lastcheckok":1,"bitrate":64}`)); d.Keep {
		t.Fatal("low bitrate should drop")
	}
	if d := Apply(s, st(`{"name":"A","url":"http://x","lastcheckok":1,"bitrate":320,"codec":"MP3"}`)); d.Keep {
		t.Fatal("excluded codec should drop")
	}
	if d := Apply(s, st(`{"name":"A","url":"http://x","lastcheckok":1,"bitrate":320,"codec":"AAC"}`)); !d.Keep {
		t.Fatal("aac should keep")
	}
}

func TestRequireHomepageAndGeo(t *testing.T) {
	s := settings.Defaults()
	s.RequireHomepage = true
	if d := Apply(s, st(`{"name":"A","url":"http://x","lastcheckok":1,"homepage":""}`)); d.Keep {
		t.Fatal("no homepage should drop")
	}
	s = settings.Defaults()
	s.RequireGeo = true
	if d := Apply(s, st(`{"name":"A","url":"http://x","lastcheckok":1}`)); d.Keep {
		t.Fatal("no geo should drop")
	}
	if d := Apply(s, st(`{"name":"A","url":"http://x","lastcheckok":1,"geo_lat":1.0,"geo_long":2.0}`)); !d.Keep {
		t.Fatal("geo should keep")
	}
}
