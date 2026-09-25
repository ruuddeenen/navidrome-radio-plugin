// SPDX-License-Identifier: GPL-3.0-or-later
//
// Package radiobrowser models radio-browser.info stations and client access.
package radiobrowser

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
)

// Station wraps the raw radio-browser JSON object with typed accessors. It is
// backed by a map so every field exposed by the API is available, including
// fields that are sometimes null.
type Station struct {
	raw map[string]any
}

// ParseStations decodes a radio-browser /json/stations/search response.
func ParseStations(data []byte) ([]Station, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var arr []map[string]any
	if err := dec.Decode(&arr); err != nil {
		return nil, err
	}
	out := make([]Station, 0, len(arr))
	for _, m := range arr {
		out = append(out, Station{raw: m})
	}
	return out, nil
}

// Raw returns the underlying map.
func (s Station) Raw() map[string]any { return s.raw }

// Str returns a field as string ("" when missing or null).
func (s Station) Str(key string) string {
	v, ok := s.raw[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case json.Number:
		return t.String()
	case bool:
		if t {
			return "true"
		}
		return "false"
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	default:
		return ""
	}
}

// Int returns a field as int (0 when missing, null or not numeric).
func (s Station) Int(key string) int {
	v, ok := s.raw[key]
	if !ok || v == nil {
		return 0
	}
	switch t := v.(type) {
	case json.Number:
		if n, err := t.Int64(); err == nil {
			return int(n)
		}
		if f, err := t.Float64(); err == nil {
			return int(f)
		}
	case float64:
		return int(t)
	case string:
		if n, err := strconv.Atoi(strings.TrimSpace(t)); err == nil {
			return n
		}
	case bool:
		if t {
			return 1
		}
	}
	return 0
}

// Float returns a field as float64.
func (s Station) Float(key string) (float64, bool) {
	v, ok := s.raw[key]
	if !ok || v == nil {
		return 0, false
	}
	switch t := v.(type) {
	case json.Number:
		if f, err := t.Float64(); err == nil {
			return f, true
		}
	case float64:
		return t, true
	case string:
		if f, err := strconv.ParseFloat(strings.TrimSpace(t), 64); err == nil {
			return f, true
		}
	}
	return 0, false
}

// Bool returns a field as bool.
func (s Station) Bool(key string) bool {
	v, ok := s.raw[key]
	if !ok || v == nil {
		return false
	}
	switch t := v.(type) {
	case bool:
		return t
	case json.Number:
		if n, err := t.Int64(); err == nil {
			return n != 0
		}
	case float64:
		return t != 0
	}
	return false
}

func splitClean(value string) []string {
	var out []string
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		part = strings.Trim(part, "[]")
		part = strings.Join(strings.Fields(part), " ")
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

// Tags returns the cleaned tag list.
func (s Station) Tags() []string { return splitClean(s.Str("tags")) }

// FirstTag returns the first non-empty tag ("" when none).
func (s Station) FirstTag() string {
	tags := s.Tags()
	if len(tags) > 0 {
		return tags[0]
	}
	return ""
}

// LanguageCodes returns the cleaned language list.
func (s Station) LanguageCodes() []string { return splitClean(s.Str("languagecodes")) }

// Field returns a logical field value for use in the name template. Unknown
// fields fall through to the raw radio-browser field with the same name, so any
// API field can be used directly.
func (s Station) Field(field string) string {
	switch field {
	case "name":
		return s.Str("name")
	case "url":
		return s.Str("url")
	case "url_resolved":
		return s.Str("url_resolved")
	case "homepage":
		return s.Str("homepage")
	case "favicon":
		return s.Str("favicon")
	case "tags":
		return strings.Join(s.Tags(), ", ")
	case "tag", "firsttag", "first_tag":
		return s.FirstTag()
	case "countrycode":
		return strings.ToUpper(strings.TrimSpace(s.Str("countrycode")))
	case "country":
		return s.Str("country")
	case "countrysubdivisioncode", "iso_3166_2":
		return s.Str("iso_3166_2")
	case "countrysubdivision", "state":
		return s.Str("state")
	case "language":
		return s.Str("language")
	case "languages":
		return strings.Join(splitClean(s.Str("language")), ", ")
	case "languagecodes":
		return strings.Join(s.LanguageCodes(), ", ")
	case "codec":
		return s.Str("codec")
	case "geoinfo":
		lat, okLat := s.Float("geo_lat")
		lon, okLon := s.Float("geo_long")
		if okLat && okLon {
			return strconv.FormatFloat(lat, 'f', 4, 64) + "," + strconv.FormatFloat(lon, 'f', 4, 64)
		}
		return ""
	default:
		return s.Str(field)
	}
}
