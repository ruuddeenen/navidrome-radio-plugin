// SPDX-License-Identifier: GPL-3.0-or-later
//
// Package template renders radio station names from a configurable template.
//
// Supported placeholders:
//
//	{field}            value of field (empty if unknown/empty)
//	{field?:default}   default when the field is empty
//
// The station exposes logical fields such as name, url, url_resolved, homepage,
// favicon, tags, tag, countrycode, country, countrysubdivisioncode,
// countrysubdivision, language, languages, languagecodes, codec, geoinfo, plus
// any raw radio-browser field as a pass-through.
package template

import (
	"regexp"
	"strings"
)

// Getter returns the value for a logical field name.
type Getter func(field string) string

var placeholder = regexp.MustCompile(`\{([a-zA-Z0-9_]+)(?:\?:([^}]*))?\}`)
var emptyGroup = regexp.MustCompile(`[\[\(\{]\s*[\]\)\}]`)

// Render substitutes the placeholders in tpl using get and normalizes the
// result (collapsed whitespace, empty bracket groups removed).
func Render(tpl string, get Getter) string {
	if strings.TrimSpace(tpl) == "" {
		tpl = "{name}"
	}
	out := placeholder.ReplaceAllStringFunc(tpl, func(match string) string {
		parts := placeholder.FindStringSubmatch(match)
		field := parts[1]
		def := ""
		if len(parts) > 2 {
			def = parts[2]
		}
		v := strings.TrimSpace(get(field))
		if v == "" {
			return def
		}
		return v
	})
	out = normalize(out)
	if out == "" {
		out = strings.TrimSpace(get("name"))
	}
	return out
}

func normalize(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	s = emptyGroup.ReplaceAllString(s, "")
	s = strings.Join(strings.Fields(s), " ")
	return strings.TrimSpace(s)
}
