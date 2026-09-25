# Automatic Radio Sync

A [Navidrome](https://www.navidrome.org/) plugin that keeps Navidrome's internet
radio stations in sync with [radio-browser.info](https://www.radio-browser.info/).

On a configurable cron schedule the plugin:

1. fetches stations from radio-browser.info (paginated, with mirror failover),
2. applies the configured filters (country, tags, language, codec, bitrate, …),
3. renders a display name from a configurable template,
4. reconciles the result against Navidrome through the Subsonic API
   (create / update / remove).

Everything runs in the background via Navidrome's task queue, so a full import
of ~55k stations never blocks the server.

> **Acknowledgements:** inspired by
> [WB2024/Add-Navidrome-Radios](https://github.com/WB2024/Add-Navidrome-Radios).
> See [Credits](#credits).

## Features

- **Configurable schedule** – a cron expression drives the sync.
- **No run on load/restart** – it only runs on the schedule, plus an optional
  one-time run after saving settings.
- **Name template** – build station names from any radio-browser field.
- **Filters** – country codes, tags, language codes and codecs
  (include/exclude), minimum bitrate and minimum votes.
- **Sensible defaults** – hides broken stations and prunes stations that no
  longer match.
- **Idempotent & resumable** – safe to re-run; a restarted server continues on
  the next run.
- **Dry run** – compute and log everything without touching Navidrome.

## How it works

Each run is split into small, retryable background tasks:

1. **plan** – fetch radio-browser pages, apply the filters and render names,
   storing the desired set in the plugin's key-value store.
2. **index** – read the existing Navidrome stations (one call) and match them by
   name.
3. **apply** – per shard: create missing stations, update changed ones, and (when
   `prune_missing` is on) remove stations that no longer match.

Navidrome enforces a case-insensitive `UNIQUE` on the station name, so the name
is the reconciliation key. Duplicate names in the desired set are deduplicated by
score (votes, clicks, bitrate, working TLS, non-HLS). This keeps runs idempotent.

## Requirements

- Navidrome **0.64 or newer** with the plugin system enabled
  (`Plugins.Enabled = true`).
- An **admin** user is required for the Subsonic calls (creating, updating and
  deleting radio stations is admin-only in Navidrome). The plugin picks the first
  admin automatically.

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

## Settings

The settings UI is grouped into four sections. See
**[docs/SETTINGS.md](docs/SETTINGS.md)** for a complete explanation of every
option, examples and edge cases.

| Setting | Section | Default | Summary |
|---|---|---|---|
| `sync_cron` | Synchronization | `30 1 * * *` | Cron expression for the recurring sync. |
| `dry_run` | Synchronization | `false` | Log the changes without writing to Navidrome. |
| `run_on_save` | Synchronization | `false` | Run a one-time sync after every settings save. |
| `name_template` | Naming | `[{countrycode?:OTHER}] [{tags}] {name}` | Builds the station name. |
| `countrycodes` + `countrycodes_mode` | Filters | (empty) / `exclude` | Include or exclude ISO country codes. |
| `tags` + `tags_mode` | Filters | (empty) / `exclude` | Include or exclude genre tags. |
| `languagecodes` + `languagecodes_mode` | Filters | (empty) / `exclude` | Include or exclude language codes. |
| `codecs` + `codecs_mode` | Filters | (empty) / `exclude` | Include or exclude audio codecs. |
| `min_bitrate` | Filters | `0` (no minimum) | Skip stations below this bitrate. |
| `min_votes` | Filters | `0` (no minimum) | Skip stations with fewer votes. |
| `hide_broken` | Behaviour | `true` | Only keep currently reachable stations. |
| `prune_missing` | Behaviour | `true` | Remove stations that no longer match. |

## Name template

The `name_template` controls the station name shown in Navidrome. A quick
example:

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

Use `{field?:default}` for a fallback when a field is empty, and note that empty
bracket groups such as `[]` are removed automatically. See
[docs/SETTINGS.md](docs/SETTINGS.md#name_template) for the full details. Names
are capped at 2048 characters.

## Station logos

Navidrome stores no images for internet radio stations. Its web player does try
to load `https://<homepage>/favicon.ico` when you play a station, so a homepage is
what makes a logo appear. The plugin therefore always fills `homePageUrl` (with a
fallback to the origin of the stream URL when the homepage is missing).

## Notes

- `prune_missing` makes the plugin manage the **entire** radio list: stations
  that are not in the filtered set — including manually added ones — are removed.
  Turn it off if you want to keep manual entries.
- Filters and the template are applied on the next run. Enabling `run_on_save`
  triggers a run immediately after saving settings.
- The radio-browser.info API is a free community service; pick a reasonable
  schedule (every 6–12 hours is fine) instead of syncing every few minutes.

## Credits

This project is inspired by
**[WB2024/Add-Navidrome-Radios](https://github.com/WB2024/Add-Navidrome-Radios)**
by [@WB2024](https://github.com/WB2024), which loads the radio-browser.info
database into Navidrome by writing directly to the database via the CLI. This
plugin takes that idea further by using the supported **Subsonic API** (no direct
database writes), with a configurable schedule, a name template and filters.

## License

GPL-3.0-or-later (uses the Navidrome plugin PDK).
