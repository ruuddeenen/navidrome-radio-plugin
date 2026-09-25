#!/usr/bin/env python3
"""Generate manifest.json for the Automatic Radio Sync plugin.

The settings are grouped into sections (JSONForms `Group`) and every field has a
`description`, which Navidrome shows as a hint under the input.
"""
from __future__ import annotations

import json
import pathlib

ROOT = pathlib.Path(__file__).resolve().parent.parent


def mode(title: str, description: str) -> dict:
    """Include/exclude toggle for a filter dimension."""
    return {
        "type": "string",
        "title": title,
        "description": description,
        "default": "exclude",
        "oneOf": [
            {"const": "exclude", "title": "Exclude"},
            {"const": "include", "title": "Include only"},
        ],
    }


def csv(title: str, description: str) -> dict:
    return {
        "type": "string",
        "title": title,
        "description": description + " Comma-separated. Empty = no filter.",
    }


BITRATES = [
    ("0", "No minimum"),
    ("32", "32 kbps"),
    ("48", "48 kbps"),
    ("64", "64 kbps"),
    ("96", "96 kbps"),
    ("128", "128 kbps"),
    ("160", "160 kbps"),
    ("192", "192 kbps"),
    ("224", "224 kbps"),
    ("256", "256 kbps"),
    ("320", "320 kbps"),
]

SYNC = [
    (
        "sync_cron",
        {
            "type": "string",
            "title": "Sync schedule (cron)",
            "description": "Cron expression: minute hour day month weekday (e.g. '30 1 * * *' = daily at 01:30).",
            "default": "30 1 * * *",
        },
    ),
    (
        "dry_run",
        {
            "type": "boolean",
            "title": "Dry run",
            "description": "Only compute and log, do not change anything in Navidrome.",
            "default": False,
        },
    ),
    (
        "sync_now",
        {
            "type": "boolean",
            "title": "Sync now (one-time)",
            "description": (
                "Runs a one-time sync after the plugin loads (i.e. after saving "
                "settings). The plugin records that it ran, so it does not run "
                "again on the next load; toggle off and on again to trigger "
                "another run. Note: the switch cannot reset itself because plugin "
                "settings are read-only from within the plugin."
            ),
            "default": False,
        },
    ),
]

NAMING = [
    (
        "name_template",
        {
            "type": "string",
            "title": "Name template",
            "description": (
                "Builds the station name. Placeholders: {name} {url} {url_resolved} "
                "{homepage} {favicon} {tags} {tag} {countrycode} {country} "
                "{countrysubdivisioncode} {countrysubdivision} {language} "
                "{languages} {languagecodes} {codec} {geoinfo}, plus any raw "
                "Radio-Browser field. Use {field?:default} for a fallback. Empty "
                "bracket groups such as [] are removed."
            ),
            "default": "[{countrycode?:OTHER}] [{tags}] {name}",
        },
    ),
]

FILTERS = [
    (
        "hide_broken",
        {
            "type": "boolean",
            "title": "Hide broken stations",
            "description": "Only include stations that are currently reachable (Radio-Browser lastcheckok).",
            "default": True,
        },
    ),
    ("countrycodes_mode", mode("Country codes", "Filter by ISO country code.")),
    ("countrycodes", csv("Country codes", "e.g. 'NL, DE, BE'.")),
    ("tags_mode", mode("Tags", "Filter by genre tags.")),
    ("tags", csv("Tags", "e.g. 'jazz, classical'.")),
    ("languagecodes_mode", mode("Language codes", "Filter by language code.")),
    ("languagecodes", csv("Language codes", "e.g. 'en, nl'.")),
    ("codecs_mode", mode("Codecs", "Filter by audio codec.")),
    ("codecs", csv("Codecs", "e.g. 'mp3, aac'.")),
    (
        "min_bitrate",
        {
            "type": "string",
            "title": "Minimum bitrate",
            "description": "Skip stations with a lower bitrate.",
            "default": "0",
            "oneOf": [{"const": v, "title": label} for v, label in BITRATES],
        },
    ),
    (
        "min_votes",
        {
            "type": "integer",
            "title": "Minimum votes",
            "description": "Skip stations with fewer votes. 0 = no minimum.",
            "default": 0,
        },
    ),
]

REQUIREMENTS = [
    (
        "require_countrycode",
        {
            "type": "boolean",
            "title": "Require country code",
            "description": "Skip stations without a country code.",
            "default": False,
        },
    ),
    (
        "require_homepage",
        {
            "type": "boolean",
            "title": "Require homepage",
            "description": "Skip stations without a homepage. The web player fetches the favicon logo from the homepage.",
            "default": False,
        },
    ),
    (
        "require_geo",
        {
            "type": "boolean",
            "title": "Require geo coordinates",
            "description": "Skip stations without geo coordinates.",
            "default": False,
        },
    ),
]

BEHAVIOUR = [
    (
        "prune_missing",
        {
            "type": "boolean",
            "title": "Remove non-matching stations",
            "description": "Removes stations that no longer match the filtered set. The plugin manages the entire radio list.",
            "default": True,
        },
    ),
]


def control(scope: str, options: dict | None = None) -> dict:
    item = {"type": "Control", "scope": f"#/properties/{scope}"}
    if options:
        item["options"] = options
    return item


def horizontal(*scopes: str) -> dict:
    return {"type": "HorizontalLayout", "elements": [control(s) for s in scopes]}


def group(label: str, elements: list[dict]) -> dict:
    return {"type": "Group", "label": label, "elements": elements}


def build() -> dict:
    properties: dict = {}
    for key, frag in SYNC + NAMING + FILTERS + REQUIREMENTS + BEHAVIOUR:
        properties[key] = frag

    ui_schema = {
        "type": "VerticalLayout",
        "elements": [
            group("Synchronization", [control("sync_cron"), control("dry_run"), control("sync_now")]),
            group("Naming", [control("name_template", {"multi": True})]),
            group(
                "Filters",
                [
                    control("hide_broken"),
                    horizontal("countrycodes_mode", "countrycodes"),
                    horizontal("tags_mode", "tags"),
                    horizontal("languagecodes_mode", "languagecodes"),
                    horizontal("codecs_mode", "codecs"),
                    horizontal("min_bitrate", "min_votes"),
                ],
            ),
            group(
                "Requirements",
                [
                    control("require_countrycode"),
                    control("require_homepage"),
                    control("require_geo"),
                ],
            ),
            group("Behaviour", [control("prune_missing")]),
        ],
    }

    return {
        "name": "Automatic Radio Sync",
        "author": "Ruud Deenen",
        "version": "0.3.0",
        "description": (
            "Periodically syncs internet radio stations from radio-browser.info into "
            "Navidrome, with configurable cron, name template and filters."
        ),
        "website": "https://github.com/ruuddeenen/navidrome-radio-sync-plugin",
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
