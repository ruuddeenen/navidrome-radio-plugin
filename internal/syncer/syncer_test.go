package syncer

import (
	"strings"
	"testing"

	"github.com/ruuddeenen/navidrome-radio-sync-plugin/internal/radiobrowser"
	"github.com/ruuddeenen/navidrome-radio-sync-plugin/internal/settings"
	"github.com/ruuddeenen/navidrome-radio-sync-plugin/internal/subsonic"
)

type memStore struct{ m map[string]string }

func newMemStore() *memStore { return &memStore{m: map[string]string{}} }

func (s *memStore) Get(key string) (string, bool) { v, ok := s.m[key]; return v, ok }
func (s *memStore) Set(key, value string)         { s.m[key] = value }
func (s *memStore) Delete(key string)             { delete(s.m, key) }
func (s *memStore) List(prefix string) ([]string, error) {
	var out []string
	for key := range s.m {
		if strings.HasPrefix(key, prefix) {
			out = append(out, key)
		}
	}
	return out, nil
}
func (s *memStore) RemovePrefix(prefix string) error {
	for key := range s.m {
		if strings.HasPrefix(key, prefix) {
			delete(s.m, key)
		}
	}
	return nil
}

type fakeFetcher struct{ pages map[int][]radiobrowser.Station }

func (f *fakeFetcher) Page(offset int) ([]radiobrowser.Station, error) {
	return f.pages[offset], nil
}

type fakeAPI struct {
	existing                  []subsonic.Radio
	creates, updates, deletes int
}

func (f *fakeAPI) List() ([]subsonic.Radio, error) { return f.existing, nil }
func (f *fakeAPI) Create(name, streamURL, homepageURL string) error {
	f.creates++
	return nil
}
func (f *fakeAPI) Update(id, name, streamURL, homepageURL string) error {
	f.updates++
	return nil
}
func (f *fakeAPI) Delete(id string) error {
	f.deletes++
	return nil
}

func station(name, url string) radiobrowser.Station {
	return stationWithVotes(name, url, 0)
}

func stationWithVotes(name, url string, votes int) radiobrowser.Station {
	body := `[{"name":"` + name + `","url":"` + url + `","url_resolved":"` + url + `","lastcheckok":1,"votes":` + itoa(votes) + `}]`
	stations, err := radiobrowser.ParseStations([]byte(body))
	if err != nil {
		panic(err)
	}
	return stations[0]
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	negative := n < 0
	if negative {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if negative {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func runAll(t *testing.T, r *Runner) {
	t.Helper()
	action, err := r.Start()
	if err != nil {
		t.Fatal(err)
	}
	for action.Kind != "" {
		switch action.Kind {
		case "plan":
			action, err = r.Plan()
		case "index":
			action, err = r.Index()
		case "apply":
			action, err = r.Apply(action.Bucket)
		default:
			t.Fatalf("unknown action %q", action.Kind)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
}

func baseSettings() settings.Settings {
	s := settings.Defaults()
	s.NameTemplate = "{name}"
	s.PageSize = 2
	return s
}

func TestCreateOnly(t *testing.T) {
	api := &fakeAPI{}
	r := &Runner{
		S:     baseSettings(),
		Store: newMemStore(),
		Fetch: &fakeFetcher{pages: map[int][]radiobrowser.Station{
			0: {station("A", "http://a"), station("B", "http://b")},
			2: {station("C", "http://c")},
		}},
		API: api,
	}
	runAll(t, r)
	if api.creates != 3 || api.updates != 0 || api.deletes != 0 {
		t.Fatalf("creates=%d updates=%d deletes=%d", api.creates, api.updates, api.deletes)
	}
	if !strings.Contains(r.Summary(), "created=3") {
		t.Fatalf("summary=%q", r.Summary())
	}
}

func TestUpdateChanged(t *testing.T) {
	api := &fakeAPI{existing: []subsonic.Radio{{ID: "1", Name: "A", StreamURL: "http://old"}}}
	r := &Runner{
		S:     baseSettings(),
		Store: newMemStore(),
		Fetch: &fakeFetcher{pages: map[int][]radiobrowser.Station{0: {station("A", "http://new")}}},
		API:   api,
	}
	runAll(t, r)
	if api.updates != 1 || api.creates != 0 || api.deletes != 0 {
		t.Fatalf("creates=%d updates=%d deletes=%d", api.creates, api.updates, api.deletes)
	}
}

func TestPruneMissing(t *testing.T) {
	api := &fakeAPI{existing: []subsonic.Radio{{ID: "9", Name: "Gone", StreamURL: "http://gone"}}}
	r := &Runner{
		S:     baseSettings(),
		Store: newMemStore(),
		Fetch: &fakeFetcher{pages: map[int][]radiobrowser.Station{}},
		API:   api,
	}
	runAll(t, r)
	if api.deletes != 1 || api.creates != 0 {
		t.Fatalf("creates=%d deletes=%d", api.creates, api.deletes)
	}
}

func TestDedupeSameName(t *testing.T) {
	api := &fakeAPI{}
	r := &Runner{
		S:     baseSettings(),
		Store: newMemStore(),
		Fetch: &fakeFetcher{pages: map[int][]radiobrowser.Station{
			0: {stationWithVotes("Dup", "http://low", 10), stationWithVotes("Dup", "http://high", 9999)},
		}},
		API: api,
	}
	runAll(t, r)
	if api.creates != 1 {
		t.Fatalf("expected a single create after dedupe, got %d", api.creates)
	}
}

func TestRemovesCaseDuplicates(t *testing.T) {
	api := &fakeAPI{existing: []subsonic.Radio{
		{ID: "1", Name: "[SK] Rádio Expres", StreamURL: "http://a"},
		{ID: "2", Name: "[SK] RÁDIO EXPRES", StreamURL: "http://b"},
	}}
	r := &Runner{
		S:     baseSettings(),
		Store: newMemStore(),
		Fetch: &fakeFetcher{pages: map[int][]radiobrowser.Station{}},
		API:   api,
	}
	runAll(t, r)
	if api.deletes != 2 {
		t.Fatalf("expected both case-variant rows removed, deletes=%d", api.deletes)
	}
}

func TestBudgetStopsEarly(t *testing.T) {
	s := baseSettings()
	s.BootstrapOpsPerRun = 1
	api := &fakeAPI{}
	r := &Runner{
		S:     s,
		Store: newMemStore(),
		Fetch: &fakeFetcher{pages: map[int][]radiobrowser.Station{
			0: {station("A", "http://a"), station("B", "http://b")},
			2: {station("C", "http://c")},
		}},
		API: api,
	}
	runAll(t, r)
	if api.creates != 1 {
		t.Fatalf("expected 1 create within budget, got %d", api.creates)
	}
}

func TestDryRunDoesNotWrite(t *testing.T) {
	s := baseSettings()
	s.DryRun = true
	api := &fakeAPI{existing: []subsonic.Radio{{ID: "9", Name: "Gone", StreamURL: "http://gone"}}}
	r := &Runner{
		S:     s,
		Store: newMemStore(),
		Fetch: &fakeFetcher{pages: map[int][]radiobrowser.Station{0: {station("A", "http://a")}}},
		API:   api,
	}
	runAll(t, r)
	if api.creates != 0 || api.updates != 0 || api.deletes != 0 {
		t.Fatalf("dry-run wrote: c=%d u=%d d=%d", api.creates, api.updates, api.deletes)
	}
}
