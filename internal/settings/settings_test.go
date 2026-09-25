package settings

import "testing"

func TestDefaults(t *testing.T) {
	s := Load(func(string) (string, bool) { return "", false })
	if s.SyncCron != "30 1 * * *" {
		t.Fatalf("cron=%q", s.SyncCron)
	}
	if s.NameTemplate != "[{countrycode?:OTHER}] [{tags}] {name}" {
		t.Fatalf("tpl=%q", s.NameTemplate)
	}
	if !s.HideBroken || !s.PruneMissing || !s.HomepageFallback {
		t.Fatal("bool defaults")
	}
	if s.BatchSize != 200 || s.PageSize != 2000 {
		t.Fatalf("int defaults: batch=%d page=%d", s.BatchSize, s.PageSize)
	}
}

func TestOverridesAndNormalization(t *testing.T) {
	m := map[string]string{
		"sync_cron":             "*/5 * * * *",
		"include_countrycodes":  "nl, de",
		"exclude_tags":          "Religious, Talk",
		"min_bitrate":           "128",
		"dry_run":               "true",
		"page_size":             "5000",
	}
	s := Load(func(k string) (string, bool) { v, ok := m[k]; return v, ok })
	if s.SyncCron != "*/5 * * * *" {
		t.Fatal(s.SyncCron)
	}
	if len(s.IncludeCountryCodes) != 2 || s.IncludeCountryCodes[0] != "NL" {
		t.Fatalf("%v", s.IncludeCountryCodes)
	}
	if len(s.ExcludeTags) != 2 || s.ExcludeTags[0] != "religious" {
		t.Fatalf("%v", s.ExcludeTags)
	}
	if s.MinBitrate != 128 || !s.DryRun || s.PageSize != 5000 {
		t.Fatalf("min=%d dry=%v page=%d", s.MinBitrate, s.DryRun, s.PageSize)
	}
}
