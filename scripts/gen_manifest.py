#!/usr/bin/env python3
"""Generate manifest.json for the Navidrome Radio Plugin.

Keeping the generator next to the code avoids drift between the Go defaults
(internal/settings) and the defaults shown in the Navidrome settings UI.
"""
from __future__ import annotations

import json
import pathlib

ROOT = pathlib.Path(__file__).resolve().parent.parent

# (key, title, json-schema fragment)
PROPERTIES: list[tuple[str, str, dict]] = [
    ("sync_cron", "Sync schedule (cron)", {"type": "string", "default": "30 1 * * *"}),
    (
        "name_template",
        "Name template",
        {
            "type": "string",
            "default": "[{countrycode?:OTHER}] [{tags}] {name}",
            "description": (
                "Placeholders: {name} {url} {url_resolved} {homepage} {favicon} {tags} "
                "{tag} {countrycode} {country} {countrysubdivisioncode} {countrysubdivision} "
                "{language} {languages} {languagecodes} {codec} {geoinfo}, plus any raw "
                "radio-browser field. Use {field?:default} for a fallback. Empty bracket "
                "groups such as [] are removed."
            ),
        },
    ),
    ("max_name_length", "Max name length (0 = unlimited)", {"type": "integer", "default": 0}),
    (
        "radiobrowser_base",
        "Custom radio-browser base URL (optional)",
        {"type": "string", "description": "When empty, the public mirrors are used with failover."},
    ),
    ("page_size", "Page size", {"type": "integer", "default": 2000}),
    ("order", "Order by (optional)", {"type": "string"}),
    ("reverse", "Reverse order", {"type": "boolean", "default": False}),
    ("hide_broken", "Hide broken stations (lastcheckok=1)", {"type": "boolean", "default": True}),
    ("require_countrycode", "Require a country code", {"type": "boolean", "default": False}),
    ("include_countrycodes", "Include country codes (comma separated)", {"type": "string"}),
    ("exclude_countrycodes", "Exclude country codes (comma separated)", {"type": "string"}),
    ("include_countries", "Include countries (comma separated)", {"type": "string"}),
    ("exclude_countries", "Exclude countries (comma separated)", {"type": "string"}),
    ("include_tags", "Include tags (comma separated)", {"type": "string"}),
    ("exclude_tags", "Exclude tags (comma separated)", {"type": "string"}),
    ("include_languages", "Include languages (comma separated)", {"type": "string"}),
    ("include_languagecodes", "Include language codes (comma separated)", {"type": "string"}),
    ("exclude_languagecodes", "Exclude language codes (comma separated)", {"type": "string"}),
    ("include_codecs", "Include codecs (comma separated)", {"type": "string"}),
    ("exclude_codecs", "Exclude codecs (comma separated)", {"type": "string"}),
    ("min_bitrate", "Minimum bitrate", {"type": "integer", "default": 0}),
    ("max_bitrate", "Maximum bitrate (0 = unlimited)", {"type": "integer", "default": 0}),
    ("exclude_hls", "Exclude HLS streams", {"type": "boolean", "default": False}),
    ("ssl_only", "Only streams without SSL errors", {"type": "boolean", "default": False}),
    ("require_homepage", "Require a homepage (for station logos)", {"type": "boolean", "default": False}),
    ("require_geo", "Require geo coordinates", {"type": "boolean", "default": False}),
    ("min_votes", "Minimum votes", {"type": "integer", "default": 0}),
    ("max_stations", "Max stations (0 = unlimited)", {"type": "integer", "default": 0}),
    ("batch_size", "Apply batch size", {"type": "integer", "default": 200}),
    ("task_delay_ms", "Delay between tasks (ms)", {"type": "integer", "default": 0}),
    (
        "bootstrap_ops_per_run",
        "Max create/update operations per run (0 = unlimited)",
        {"type": "integer", "default": 0},
    ),
    ("prune_missing", "Remove stations no longer matching", {"type": "boolean", "default": True}),
    ("dry_run", "Dry run (do not write to Navidrome)", {"type": "boolean", "default": False}),
    ("admin_user", "Admin user for Subsonic calls (empty = first admin)", {"type": "string"}),
    ("homepage_fallback", "Use stream URL as homepage fallback", {"type": "boolean", "default": True}),
]


def control(prop: str, options: dict | None = None) -> dict:
    item = {"type": "Control", "scope": f"#/properties/{prop}"}
    if options:
        item["options"] = options
    return item


def build() -> dict:
    properties = {key: frag for key, _title, frag in PROPERTIES}
    elements = []
    for key, _title, _frag in PROPERTIES:
        opts = None
        if key in ("include_countrycodes", "exclude_countrycodes",
                   "include_countries", "exclude_countries",
                   "include_tags", "exclude_tags", "include_languages",
                   "include_languagecodes", "exclude_languagecodes",
                   "include_codecs", "exclude_codecs"):
            opts = {"multi": True}
        if key == "name_template":
            opts = {"multi": True}
        elements.append(control(key, opts))

    ui_schema = {"type": "VerticalLayout", "elements": elements}

    return {
        "name": "Navidrome Radio Plugin",
        "author": "Ruud Deenen",
        "version": "0.1.0",
        "description": (
            "Periodically syncs internet radio stations from radio-browser.info into "
            "Navidrome, with configurable cron, name template and filters."
        ),
        "website": "https://github.com/ruuddeenen/navidrome-radio-plugin",
        "config": {
            "schema": {"type": "object", "properties": properties, "required": []},
            "uiSchema": ui_schema,
        },
        "permissions": {
            "http": {
                "reason": "Fetch stations from radio-browser.info.",
                "requiredHosts": [
                    "de1.api.radio-browser.info",
                    "de2.api.radio-browser.info",
                    "nl1.api.radio-browser.info",
                    "at1.api.radio-browser.info",
                    "fr1.api.radio-browser.info",
                    "all.api.radio-browser.info",
                    "*.radio-browser.info",
                ],
            },
            "kvstore": {
                "reason": "Persist sync progress and the station ID mapping.",
                "maxSize": "64MB",
            },
            "scheduler": {"reason": "Run the periodic radio sync."},
            "taskqueue": {
                "reason": "Process stations in the background with retries.",
                "maxConcurrency": 2,
            },
            "subsonicapi": {"reason": "Create, update and delete internet radio stations."},
            "users": {"reason": "Authorize the internal Subsonic API calls."},
        },
    }


def main() -> None:
    manifest = build()
    out = ROOT / "manifest.json"
    out.write_text(json.dumps(manifest, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
    print(f"wrote {out}")


if __name__ == "__main__":
    main()
