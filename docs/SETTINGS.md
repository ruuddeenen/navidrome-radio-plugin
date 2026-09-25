# Settings reference

This document explains every setting of the **Automatic Radio Sync** plugin in
detail. The same information (as short hints) is shown under each field in the
Navidrome **Settings → Plugins** UI.

Settings are grouped into four sections:

| Section | Settings |
|---|---|
| **Synchronization** | `sync_cron`, `dry_run`, `run_on_save` |
| **Naming** | `name_template` |
| **Filters** | `countrycodes`, `tags`, `languagecodes`, `codecs`, `min_bitrate`, `min_votes` |
| **Behaviour** | `hide_broken`, `prune_missing` |

Two rules apply to every run, independent of the settings:

- a station must have a **name** and a **stream URL**, otherwise it is skipped;
- the desired set is **deduplicated by station name** (Navidrome's name is
  unique). When several stations produce the same name, the one with the highest
  score wins (votes, clicks, bitrate, working TLS, non-HLS).

Changes take effect on the next run. Saving settings reloads the plugin, so you
can use `run_on_save` to trigger that run immediately.

---

## Synchronization

### `sync_cron`

- **Type:** string (cron expression)
- **Default:** `30 1 * * *`
- **Description:** when the recurring sync runs.

Standard 5-field cron: `minute hour day-of-month month day-of-week`. Examples:

| Value | Meaning |
|---|---|
| `30 1 * * *` | every day at 01:30 |
| `0 */6 * * *` | every 6 hours (00:00, 06:00, 12:00, 18:00) |
| `0 3 * * 0` | every Sunday at 03:00 |

The plugin does **not** sync when Navidrome starts or restarts. Only the schedule
(and optionally `run_on_save`) triggers a run. The radio-browser.info API is a
free community service, so prefer a modest frequency (every 6–12 hours is
plenty) over running every few minutes.

### `dry_run`

- **Type:** boolean
- **Default:** `false`
- **Description:** compute and log the changes without modifying Navidrome.

With `dry_run` on, the plugin still fetches, filters and reconciles, and logs what
it *would* create, update or delete, but makes no changes and does not prune
stations. Useful for testing a new template or filter set.

### `run_on_save`

- **Type:** boolean
- **Default:** `false`
- **Description:** run a one-time sync after every settings save.

Saving plugin settings unloads and reloads the plugin; on load the plugin
compares a fingerprint of the current configuration with the previous one and, if
they differ and `run_on_save` is enabled, starts one sync. Because the comparison
is based on the configuration itself:

- a **restart** without a configuration change does **not** trigger a run;
- every time a **changed** configuration is saved, exactly **one** sync runs.

Leave it on to sync after every change, or turn it off (default) to only rely on
`sync_cron`.

---

## Naming

### `name_template`

- **Type:** string
- **Default:** `[{countrycode?:OTHER}] [{tags}] {name}`
- **Description:** builds the station name that Navidrome displays.

The template is rendered per station using radio-browser fields. Rendered names
are capped at **2048 characters** (Navidrome does not enforce a limit of its own,
so this is a safety bound; the longest names in a full radio-browser dump are
around 1.250 characters).

#### Syntax

| Syntax | Meaning |
|---|---|
| `{field}` | value of `field`; expands to an empty string when missing |
| `{field?:default}` | value of `field`, or `default` when the field is empty |

After substitution, whitespace is collapsed and empty bracket groups (`[]`,
`()`, `{}`) are removed, so an empty `{tags}` does not leave a stray `[]`.

#### Placeholders

| Placeholder | Source | Example |
|---|---|---|
| `{name}` | `name` | `Arrow Classic Rock` |
| `{url}` | `url` | `http://stream.gal.io/arrow` |
| `{url_resolved}` | `url_resolved` | `http://stream.gal.io/arrow` |
| `{homepage}` | `homepage` | `https://www.arrow.nl/` |
| `{favicon}` | `favicon` | `https://www.arrow.nl/favicon.ico` |
| `{tags}` | `tags` (cleaned, comma separated) | `classic rock, pop rock` |
| `{tag}` / `{firsttag}` | first tag | `classic rock` |
| `{countrycode}` | `countrycode` (upper case) | `NL` |
| `{country}` | `country` | `The Netherlands` |
| `{countrysubdivisioncode}` / `{iso_3166_2}` | `iso_3166_2` | `NL-ZH` |
| `{countrysubdivision}` / `{state}` | `state` | `Zuid-Holland` |
| `{language}` | `language` | `dutch` |
| `{languages}` | `language` (comma separated) | `dutch, english` |
| `{languagecodes}` | `languagecodes` (comma separated) | `NL, EN` |
| `{codec}` | `codec` | `MP3` |
| `{geoinfo}` | `geo_lat`,`geo_long` | `52.0000,4.0000` |
| *anything else* | raw radio-browser field with that name | e.g. `{bitrate}`, `{votes}`, `{clickcount}` |

Any field returned by the radio-browser API can be used directly; unknown fields
simply expand to an empty string.

#### Examples

| Template | Result |
|---|---|
| `[{countrycode?:OTHER}] [{tags}] {name}` | `[NL] [classic rock] Arrow Classic Rock` |
| `{name} ({countrycode?:XX})` | `Arrow Classic Rock (NL)` |
| `[{countrycode?:OTHER}] {name}` | `[DE] bigFM RnB` |
| `{tags?:no genre} – {name}` | `blasmusik, folk – Blasmusikradio` |

Note that the name is the unique key in Navidrome, so two stations that render to
the same name will collapse into one (the best-scoring one).

---

## Filters

Each of the four list filters (country codes, tags, language codes, codecs) has
two fields in the UI:

- a **mode** toggle: `Exclude` or `Include only`;
- a **comma-separated list**.

Behaviour for all four is the same:

- an **empty list disables the filter** (regardless of the mode);
- mode **`Exclude`** (default): drop stations that match any value in the list;
- mode **`Include only`**: keep only stations that match at least one value in
  the list.

Matching is case-insensitive: country codes are compared upper-cased, the other
values lower-cased.

### `countrycodes` + `countrycodes_mode`

- **Default:** empty list, mode `exclude`
- **Matches:** the station's ISO `countrycode` (e.g. `NL`, `DE`, `US`).
- **Example (include only):** `NR, DE, BE` keeps only stations from those
  countries.

### `tags` + `tags_mode`

- **Default:** empty list, mode `exclude`
- **Matches:** any of the station's comma-separated tags (genre keywords from
  radio-browser, free text).
- **Example (exclude):** `religious, talk` drops stations tagged with either.
- **Example (include only):** `jazz, soul` keeps stations tagged jazz or soul.

### `languagecodes` + `languagecodes_mode`

- **Default:** empty list, mode `exclude`
- **Matches:** any of the station's language codes (e.g. `EN`, `NL`, `DE`).
- **Example (include only):** `NL, EN` keeps Dutch- and English-language
  stations.

### `codecs` + `codecs_mode`

- **Default:** empty list, mode `exclude`
- **Matches:** the station's audio codec (e.g. `MP3`, `AAC`, `OGG`), compared
  case-insensitively.
- **Example (exclude):** `mp3` drops MP3 streams.
- **Example (include only):** `aac, ogg` keeps only AAC/OGG streams.

### `min_bitrate`

- **Type:** dropdown
- **Default:** `No minimum`
- **Description:** skip stations with a lower bitrate (kbps).

Available values: `No minimum`, 32, 48, 64, 96, 128, 160, 192, 224, 256, 320 kbps.
`No minimum` (0) disables the filter.

### `min_votes`

- **Type:** integer
- **Default:** `0` (no minimum)
- **Description:** skip stations with fewer votes than this value.

Useful to keep only stations that the radio-browser community has validated.

---

## Behaviour

### `hide_broken`

- **Type:** boolean
- **Default:** `true`
- **Description:** only include stations that are currently reachable.

radio-browser marks each station with a `lastcheckok` status. With this enabled,
stations whose last check failed are excluded. Recommended to leave this on.

### `prune_missing`

- **Type:** boolean
- **Default:** `true`
- **Description:** remove stations that no longer match the filtered set.

When enabled, the plugin manages the **entire** radio list: after matching the
desired set, any remaining station in Navidrome is deleted. This cleans up
stations that disappeared from radio-browser, changed country/tag and fell out of
your filters, or were added manually and are not in radio-browser.

Turn it **off** if you want to keep manually added stations and only ever add or
update — never delete. Deletion is always skipped during `dry_run`.

---

## Fixed internal values

These are not configurable (they are built in with sensible values):

| Value | Setting |
|---|---|
| Page size when fetching radio-browser | 2000 (kept below the 10 MB HTTP response limit) |
| radio-browser mirrors | the public `de1`/`de2`/`nl1`/`at1`/`fr1` mirrors, tried in order on failure |
| Maximum rendered name length | 2048 characters |
| Homepage fallback | the origin of the stream URL is used when a station has no homepage |
| Admin user for Subsonic calls | the first admin, automatically |
| Duplicate-name resolution | highest score (votes, clicks, bitrate, TLS, non-HLS) |

## Credits

The plugin is inspired by
[WB2024/Add-Navidrome-Radios](https://github.com/WB2024/Add-Navidrome-Radios).
