// SPDX-License-Identifier: GPL-3.0-or-later
//
// Package settings loads and represents the plugin configuration. It is kept
// free of any host/PDK dependency so it can be unit tested on any platform.
package settings

import (
	"strconv"
	"strings"
)

// MaxNameLength is the fixed upper bound for a rendered station name. Navidrome
// stores `name` as an unbounded varchar, so this is a safety bound only; the
// longest station name observed is ~1.255 characters.
const MaxNameLength = 2048

// pageSize is the fixed radio-browser page size. Kept below the 10 MB outbound
// HTTP response limit.
const pageSize = 2000

// Getter retrieves a raw configuration value by key.
type Getter func(key string) (string, bool)

// Settings is the fully resolved plugin configuration.
type Settings struct {
	// Schedule (cron) and naming.
	SyncCron      string
	NameTemplate  string
	MaxNameLength int

	// radio-browser source (not user configurable).
	BaseURL  string
	PageSize int
	Order    string
	Reverse  bool

	// Filters.
	HideBroken           bool
	IncludeCountryCodes  []string
	ExcludeCountryCodes  []string
	IncludeTags          []string
	ExcludeTags          []string
	IncludeLanguageCodes []string
	ExcludeLanguageCodes []string
	IncludeCodecs        []string
	ExcludeCodecs        []string
	MinBitrate           int
	MinVotes             int

	// Limits.
	MaxStations int

	// Sync behaviour.
	BatchSize          int
	TaskDelayMs        int
	BootstrapOpsPerRun int
	PruneMissing       bool
	DryRun             bool
	RunOnSave          bool
	AdminUser          string
	HomepageFallback   bool
}

// Defaults returns the built-in defaults.
func Defaults() Settings {
	return Settings{
		SyncCron:         "30 1 * * *",
		NameTemplate:     "[{countrycode?:OTHER}] [{tags}] {name}",
		MaxNameLength:    MaxNameLength,
		PageSize:         pageSize,
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

	s.HideBroken = boolean(get, "hide_broken", s.HideBroken)

	applyMode(&s.IncludeCountryCodes, &s.ExcludeCountryCodes,
		value(get, "countrycodes_mode", "exclude"), upperList(value(get, "countrycodes", "")))
	applyMode(&s.IncludeTags, &s.ExcludeTags,
		value(get, "tags_mode", "exclude"), lowerList(value(get, "tags", "")))
	applyMode(&s.IncludeLanguageCodes, &s.ExcludeLanguageCodes,
		value(get, "languagecodes_mode", "exclude"), lowerList(value(get, "languagecodes", "")))
	applyMode(&s.IncludeCodecs, &s.ExcludeCodecs,
		value(get, "codecs_mode", "exclude"), lowerList(value(get, "codecs", "")))

	s.MinBitrate = integer(get, "min_bitrate", 0)
	s.MinVotes = integer(get, "min_votes", 0)

	s.PruneMissing = boolean(get, "prune_missing", s.PruneMissing)
	s.DryRun = boolean(get, "dry_run", false)
	s.RunOnSave = boolean(get, "run_on_save", false)

	return s
}

// applyMode routes the values to the include or exclude list based on the
// include/exclude toggle (default: exclude).
func applyMode(include, exclude *[]string, mode string, values []string) {
	if len(values) == 0 {
		return
	}
	if strings.EqualFold(strings.TrimSpace(mode), "include") {
		*include = values
	} else {
		*exclude = values
	}
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
