package radiobrowser

import "testing"

const sample = `[{"name":"Radio Paradise","url":"http://a","url_resolved":"http://b","homepage":null,` +
	`"favicon":"https://f.ico","tags":"california,eclectic","countrycode":"us",` +
	`"country":"The United States","iso_3166_2":"US-CA","state":"California",` +
	`"language":"english","languagecodes":"EN","codec":"AAC","bitrate":320,"votes":317298,` +
	`"lastcheckok":1,"geo_lat":40.785114,"geo_long":-124.183922,"hls":0,"ssl_error":0,"serveruuid":null}]`

func TestParseAndFields(t *testing.T) {
	stations, err := ParseStations([]byte(sample))
	if err != nil {
		t.Fatal(err)
	}
	if len(stations) != 1 {
		t.Fatalf("len=%d", len(stations))
	}
	st := stations[0]
	if st.Str("homepage") != "" {
		t.Fatalf("null homepage should be empty, got %q", st.Str("homepage"))
	}
	if st.Field("countrycode") != "US" {
		t.Fatalf("countrycode=%q", st.Field("countrycode"))
	}
	if st.Field("tags") != "california, eclectic" {
		t.Fatalf("tags=%q", st.Field("tags"))
	}
	if st.Field("tag") != "california" {
		t.Fatalf("tag=%q", st.Field("tag"))
	}
	if st.Int("bitrate") != 320 {
		t.Fatalf("bitrate=%d", st.Int("bitrate"))
	}
	if got := st.Field("geoinfo"); got != "40.7851,-124.1839" {
		t.Fatalf("geoinfo=%q", got)
	}
	if st.Field("countrysubdivisioncode") != "US-CA" {
		t.Fatalf("iso=%q", st.Field("countrysubdivisioncode"))
	}
	if st.Field("url_resolved") != "http://b" {
		t.Fatalf("url_resolved=%q", st.Field("url_resolved"))
	}
}
