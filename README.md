# Automatic Radio Sync

A [Navidrome](https://www.navidrome.org/) plugin that keeps Navidrome's internet
radio stations in sync with [radio-browser.info](https://www.radio-browser.info/).

On a configurable cron schedule the plugin:

1. fetches stations from radio-browser.info (paginated, with mirror failover),
2. applies the configured filters (country, tags, language, codec, bitrate, …),
3. renders a display name from a configurable template,
4. creates / updates / removes the matching stations in Navidrome through the
   Subsonic API.

> **Status:** work in progress. Fetch, filter, template, plan/index/apply
> reconciliation (create/update/delete) and the dry-run work. Throttling and the
> full initial import are being validated.

## Requirements

- Navidrome **0.64 or newer** with the plugin system enabled (`Plugins.Enabled = true`).
- The plugin needs an **admin** user for its Subsonic calls (create/update/delete
  radio stations are admin-only in Navidrome).

## How station names are built

The `name_template` controls the station name shown in Navidrome:

```
[{countrycode?:OTHER}] [{tags}] {name}
```

| Placeholder | Value |
|---|---|
| `{name}` | station name |
| `{url}` / `{url_resolved}` | stream URL (raw / resolved) |
| `{homepage}` | station homepage |
| `{favicon}` | favicon URL from radio-browser |
| `{tags}` / `{tag}` | full tag list / first tag |
| `{countrycode}` / `{country}` | ISO country code / country name |
| `{countrysubdivisioncode}` / `{countrysubdivision}` | `iso_3166_2` / `state` |
| `{language}` / `{languages}` / `{languagecodes}` | language fields |
| `{codec}` | audio codec |
| `{geoinfo}` | `lat,lon` when available |
| anything else | raw radio-browser field with the same name |

Use `{field?:default}` for a fallback when the field is empty. Empty bracket
groups such as `[]` are removed automatically.

## Station logos

Navidrome stores no images for internet radio stations. Its web player does try
to load `https://<homepage>/favicon.ico` when you play a station, so setting a
homepage is what makes a logo appear. The plugin therefore always fills
`homePageUrl` (with an optional fallback to the stream URL).

## Settings

The settings UI is grouped into sections:

- **Synchronization** – `sync_cron`, `dry_run`, `run_on_save` (when enabled,
  runs a one-time sync whenever the settings are saved; default off).
- **Naming** – `name_template`.
- **Filters** – `hide_broken`; for each of country codes, tags, language codes
  and codecs an include/exclude toggle plus a comma-separated list (empty = no
  filter); `min_bitrate` and `min_votes`.
- **Requirements** – `require_countrycode`, `require_homepage`, `require_geo`.
- **Behaviour** – `prune_missing`.

Rendered station names are capped at 2048 characters.

## Build

The Navidrome plugin PDK is not published as a versioned Go module, so the build
clones Navidrome at a pinned version and points `go.mod`'s `replace` at it.

```bash
make deps              # clone Navidrome into .build/navidrome
make test              # unit tests
make build-go          # standard Go wasip1 build -> automatic-radio-sync.ndp
# or, inside a container (hosts without Go):
make build-go-docker
make test-go-docker
```

## Install

1. Copy `automatic-radio-sync.ndp` to `<DataFolder>/plugins/`.
2. Ensure `Plugins.Enabled = true` in `navidrome.toml`.
3. Enable the plugin and grant access:
   ```bash
   navidrome plugin enable automatic-radio-sync
   navidrome plugin edit automatic-radio-sync \
     --all-users \
     --config '{"sync_cron":"30 1 * * *"}'
   ```
   Or use the Navidrome **Settings → Plugins** UI.
4. Restart Navidrome so the running server picks up the enabled plugin.

## License

GPL-3.0-or-later (uses the Navidrome plugin PDK).
