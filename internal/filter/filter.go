// SPDX-License-Identifier: GPL-3.0-or-later
//
// Package filter decides whether a radio-browser station matches the configured
// filters. It is pure and unit tested.
package filter

import (
	"strings"

	"github.com/ruuddeenen/navidrome-radio-plugin/internal/radiobrowser"
	"github.com/ruuddeenen/navidrome-radio-plugin/internal/settings"
)

// Decision is the filter outcome.
type Decision struct {
	Keep   bool
	Reason string
}

// Apply evaluates all configured filters against a station.
func Apply(s settings.Settings, st radiobrowser.Station) Decision {
	if strings.TrimSpace(st.Str("name")) == "" {
		return reject("no_name")
	}
	if settings.EffectiveURL(st.Str("url"), st.Str("url_resolved")) == "" {
		return reject("no_url")
	}

	if s.HideBroken && st.Int("lastcheckok") != 1 {
		return reject("broken")
	}

	code := strings.ToUpper(strings.TrimSpace(st.Str("countrycode")))
	if s.RequireCountryCode && code == "" {
		return reject("no_countrycode")
	}
	if len(s.IncludeCountryCodes) > 0 && !contains(s.IncludeCountryCodes, code) {
		return reject("countrycode_not_included")
	}
	if contains(s.ExcludeCountryCodes, code) {
		return reject("countrycode_excluded")
	}

	tags := lowerSet(st.Tags())
	if len(s.IncludeTags) > 0 && !intersects(s.IncludeTags, tags) {
		return reject("tag_not_included")
	}
	if intersects(s.ExcludeTags, tags) {
		return reject("tag_excluded")
	}

	languageCodes := lowerSet(st.LanguageCodes())
	if len(s.IncludeLanguageCodes) > 0 && !intersects(s.IncludeLanguageCodes, languageCodes) {
		return reject("languagecode_not_included")
	}
	if intersects(s.ExcludeLanguageCodes, languageCodes) {
		return reject("languagecode_excluded")
	}

	codec := strings.ToLower(strings.TrimSpace(st.Str("codec")))
	if len(s.IncludeCodecs) > 0 && !contains(s.IncludeCodecs, codec) {
		return reject("codec_not_included")
	}
	if contains(s.ExcludeCodecs, codec) {
		return reject("codec_excluded")
	}

	if s.MinBitrate > 0 && st.Int("bitrate") < s.MinBitrate {
		return reject("bitrate_below_min")
	}

	if s.MinVotes > 0 && st.Int("votes") < s.MinVotes {
		return reject("votes_below_min")
	}
	if s.RequireHomepage && strings.TrimSpace(st.Str("homepage")) == "" {
		return reject("no_homepage")
	}
	if s.RequireGeo {
		if _, ok := st.Float("geo_lat"); !ok {
			return reject("no_geo")
		}
		if _, ok := st.Float("geo_long"); !ok {
			return reject("no_geo")
		}
	}

	return Decision{Keep: true}
}

func reject(reason string) Decision { return Decision{Keep: false, Reason: reason} }

func contains(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

func lowerSet(values []string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.ToLower(strings.TrimSpace(v))
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

func intersects(a, b []string) bool {
	for _, item := range a {
		if contains(b, item) {
			return true
		}
	}
	return false
}
