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
	if s.MaxNameLength != 2048 {
		t.Fatalf("max name length=%d", s.MaxNameLength)
	}
	if !s.HideBroken || !s.PruneMissing || !s.HomepageFallback {
		t.Fatal("bool defaults")
	}
	if s.BatchSize != 200 || s.PageSize != 2000 {
		t.Fatalf("int defaults: batch=%d page=%d", s.BatchSize, s.PageSize)
	}
}

func TestOverridesAndModeMapping(t *testing.T) {
	m := map[string]string{
		"sync_cron":           "*/5 * * * *",
		"countrycodes_mode":   "include",
		"countrycodes":        "nl, de",
		"tags_mode":           "exclude",
		"tags":                "Religious, Talk",
		"languagecodes_mode":  "include",
		"languagecodes":       "en, nl",
		"codecs_mode":         "exclude",
		"codecs":              "mp3",
		"min_bitrate":         "128",
		"min_votes":           "25",
		"require_homepage":    "true",
		"dry_run":             "true",
	}
	s := Load(func(k string) (string, bool) { v, ok := m[k]; return v, ok })

	if s.SyncCron != "*/5 * * * *" {
		t.Fatal(s.SyncCron)
	}
	// include mode -> Include list, normalized to upper case for country codes.
	if len(s.IncludeCountryCodes) != 2 || s.IncludeCountryCodes[0] != "NL" || len(s.ExcludeCountryCodes) != 0 {
		t.Fatalf("countrycodes: include=%v exclude=%v", s.IncludeCountryCodes, s.ExcludeCountryCodes)
	}
	// exclude mode -> Exclude list, lower cased.
	if len(s.ExcludeTags) != 2 || s.ExcludeTags[0] != "religious" || len(s.IncludeTags) != 0 {
		t.Fatalf("tags: include=%v exclude=%v", s.IncludeTags, s.ExcludeTags)
	}
	if len(s.IncludeLanguageCodes) != 2 || s.IncludeLanguageCodes[0] != "en" {
		t.Fatalf("languagecodes: %v", s.IncludeLanguageCodes)
	}
	if len(s.ExcludeCodecs) != 1 || s.ExcludeCodecs[0] != "mp3" {
		t.Fatalf("codecs: %v", s.ExcludeCodecs)
	}
	if s.MinBitrate != 128 || s.MinVotes != 25 || !s.RequireHomepage || !s.DryRun {
		t.Fatalf("min=%d votes=%d hp=%v dry=%v", s.MinBitrate, s.MinVotes, s.RequireHomepage, s.DryRun)
	}
}

func TestEmptyFilterLists(t *testing.T) {
	m := map[string]string{"countrycodes_mode": "include", "countrycodes": ""}
	s := Load(func(k string) (string, bool) { v, ok := m[k]; return v, ok })
	if len(s.IncludeCountryCodes) != 0 || len(s.ExcludeCountryCodes) != 0 {
		t.Fatalf("empty list should not enable a filter: inc=%v exc=%v", s.IncludeCountryCodes, s.ExcludeCountryCodes)
	}
}
