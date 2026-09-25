// SPDX-License-Identifier: GPL-3.0-or-later
//
// Package settings loads and represents the plugin configuration. It is kept
// free of any host/PDK dependency so it can be unit tested on any platform.
package settings

import (
	"strconv"
	"strings"
)

// Getter retrieves a raw configuration value by key.
type Getter func(key string) (string, bool)

// Settings is the fully resolved plugin configuration.
type Settings struct {
	// Schedule (cron) and naming.
	SyncCron      string
	NameTemplate  string
	MaxNameLength int

	// radio-browser source.
	BaseURL    string
	PageSize   int
	Order      string
	Reverse    bool
	HideBroken bool

	// Filters.
	RequireCountryCode   bool
	IncludeCountryCodes  []string
	ExcludeCountryCodes  []string
	IncludeCountries     []string
	ExcludeCountries     []string
	IncludeTags          []string
	ExcludeTags          []string
	IncludeLanguages     []string
	IncludeLanguageCodes []string
	ExcludeLanguageCodes []string
	IncludeCodecs        []string
	ExcludeCodecs        []string
	MinBitrate           int
	MaxBitrate           int
	ExcludeHLS           bool
	SSLOnly              bool
	RequireHomepage      bool
	RequireGeo           bool
	MinVotes             int

	// Limits.
	MaxStations int

	// Sync behaviour.
	BatchSize          int
	TaskDelayMs        int
	BootstrapOpsPerRun int
	PruneMissing       bool
	DryRun             bool
	AdminUser          string
	HomepageFallback   bool
}

// Defaults returns the built-in defaults.
func Defaults() Settings {
	return Settings{
		SyncCron:         "30 1 * * *",
		NameTemplate:     "[{countrycode?:OTHER}] [{tags}] {name}",
		PageSize:         2000,
		HideBroken:       true,
		BatchSize:        200,
		PruneMissing:     true,
		HomepageFallback: true,
	}
}

// Load reads the configuration via the getter, falling back to defaults.
func Load(get Getter) Settings {
	s := Defaults()

	s.SyncCron = value(get, "sync_cron", s.SyncCron)
	s.NameTemplate = value(get, "name_template", s.NameTemplate)
	s.MaxNameLength = integer(get, "max_name_length", 0)

	s.BaseURL = strings.TrimSpace(value(get, "radiobrowser_base", ""))
	if v := integer(get, "page_size", 0); v > 0 {
		s.PageSize = v
	}
	s.Order = strings.TrimSpace(value(get, "order", ""))
	s.Reverse = boolean(get, "reverse", false)
	s.HideBroken = boolean(get, "hide_broken", s.HideBroken)

	s.RequireCountryCode = boolean(get, "require_countrycode", false)
	s.IncludeCountryCodes = upperList(value(get, "include_countrycodes", ""))
	s.ExcludeCountryCodes = upperList(value(get, "exclude_countrycodes", ""))
	s.IncludeCountries = lowerList(value(get, "include_countries", ""))
	s.ExcludeCountries = lowerList(value(get, "exclude_countries", ""))
	s.IncludeTags = lowerList(value(get, "include_tags", ""))
	s.ExcludeTags = lowerList(value(get, "exclude_tags", ""))
	s.IncludeLanguages = lowerList(value(get, "include_languages", ""))
	s.IncludeLanguageCodes = lowerList(value(get, "include_languagecodes", ""))
	s.ExcludeLanguageCodes = lowerList(value(get, "exclude_languagecodes", ""))
	s.IncludeCodecs = lowerList(value(get, "include_codecs", ""))
	s.ExcludeCodecs = lowerList(value(get, "exclude_codecs", ""))
	s.MinBitrate = integer(get, "min_bitrate", 0)
	s.MaxBitrate = integer(get, "max_bitrate", 0)
	s.ExcludeHLS = boolean(get, "exclude_hls", false)
	s.SSLOnly = boolean(get, "ssl_only", false)
	s.RequireHomepage = boolean(get, "require_homepage", false)
	s.RequireGeo = boolean(get, "require_geo", false)
	s.MinVotes = integer(get, "min_votes", 0)

	s.MaxStations = integer(get, "max_stations", 0)
	if v := integer(get, "batch_size", 0); v > 0 {
		s.BatchSize = v
	}
	s.TaskDelayMs = integer(get, "task_delay_ms", 0)
	s.BootstrapOpsPerRun = integer(get, "bootstrap_ops_per_run", 0)
	s.PruneMissing = boolean(get, "prune_missing", s.PruneMissing)
	s.DryRun = boolean(get, "dry_run", false)
	s.AdminUser = strings.TrimSpace(value(get, "admin_user", ""))
	s.HomepageFallback = boolean(get, "homepage_fallback", s.HomepageFallback)

	return s
}

// EffectiveURL returns the stream URL to use for a station (resolved first).
func EffectiveURL(raw string, resolved string) string {
	if strings.TrimSpace(resolved) != "" {
		return strings.TrimSpace(resolved)
	}
	return strings.TrimSpace(raw)
}

func value(get Getter, key, fallback string) string {
	if v, ok := get(key); ok {
		return v
	}
	return fallback
}

func boolean(get Getter, key string, fallback bool) bool {
	v, ok := get(key)
	if !ok {
		return fallback
	}
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off", "":
		return false
	default:
		return fallback
	}
}

func integer(get Getter, key string, fallback int) int {
	v, ok := get(key)
	if !ok {
		return fallback
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return fallback
	}
	return parsed
}

func splitList(value string) []string {
	var out []string
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func lowerList(value string) []string {
	items := splitList(value)
	for i := range items {
		items[i] = strings.ToLower(items[i])
	}
	return items
}

func upperList(value string) []string {
	items := splitList(value)
	for i := range items {
		items[i] = strings.ToUpper(items[i])
	}
	return items
}
