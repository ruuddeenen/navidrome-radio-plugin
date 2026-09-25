#!/usr/bin/env python3
"""Generate manifest.json for the Navidrome Radio Sync Plugin.

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
            {"const": "exclude", "title": "Uitsluiten"},
            {"const": "include", "title": "Alleen deze"},
        ],
    }


def csv(title: str, description: str) -> dict:
    return {
        "type": "string",
        "title": title,
        "description": description + " Komma-gescheiden. Leeg = geen filter.",
    }


BITRATES = [
    ("0", "Geen minimum"),
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
            "title": "Sync-planning (cron)",
            "description": "Cron-expressie: minuut uur dag maand weekdag (bijv. '30 1 * * *' = dagelijks 01:30).",
            "default": "30 1 * * *",
        },
    ),
    (
        "dry_run",
        {
            "type": "boolean",
            "title": "Dry run",
            "description": "Alleen berekenen en loggen, niets aanpassen in Navidrome.",
            "default": False,
        },
    ),
]

NAMING = [
    (
        "name_template",
        {
            "type": "string",
            "title": "Naam-template",
            "description": (
                "Bepaalt de zendernaam. Placeholders: {name} {url} {url_resolved} "
                "{homepage} {favicon} {tags} {tag} {countrycode} {country} "
                "{countrysubdivisioncode} {countrysubdivision} {language} "
                "{languages} {languagecodes} {codec} {geoinfo}, plus elk ruw "
                "Radio-Browser-veld. Gebruik {veld?:standaard} voor een terugval. "
                "Lege haakjes zoals [] worden verwijderd."
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
            "title": "Kapotte zenders verbergen",
            "description": "Alleen zenders die op dit moment bereikbaar zijn (Radio-Browser lastcheckok).",
            "default": True,
        },
    ),
    ("countrycodes_mode", mode("Landcodes", "Filter op ISO-landcode. Kies 'Uitsluiten' of 'Alleen deze'.")),
    ("countrycodes", csv("Landcodes", "Bijv. 'NL, DE, BE'.")),
    ("tags_mode", mode("Tags", "Filter op genre-tags.")),
    ("tags", csv("Tags", "Bijv. 'jazz, classical'.")),
    ("languagecodes_mode", mode("Taalcodes", "Filter op taalcode.")),
    ("languagecodes", csv("Taalcodes", "Bijv. 'en, nl'.")),
    ("codecs_mode", mode("Codecs", "Filter op audiocodec.")),
    ("codecs", csv("Codecs", "Bijv. 'mp3, aac'.")),
    (
        "min_bitrate",
        {
            "type": "string",
            "title": "Minimum bitrate",
            "description": "Sla zenders met een lagere bitrate over.",
            "default": "0",
            "oneOf": [{"const": v, "title": label} for v, label in BITRATES],
        },
    ),
    (
        "min_votes",
        {
            "type": "integer",
            "title": "Minimum votes",
            "description": "Sla zenders met minder stemmen over. 0 = geen minimum.",
            "default": 0,
        },
    ),
]

REQUIREMENTS = [
    (
        "require_countrycode",
        {
            "type": "boolean",
            "title": "Landcode vereist",
            "description": "Sla zenders zonder landcode over.",
            "default": False,
        },
    ),
    (
        "require_homepage",
        {
            "type": "boolean",
            "title": "Homepage vereist",
            "description": "Sla zenders zonder homepage over. De webspeler haalt het favicon-logo van de homepage.",
            "default": False,
        },
    ),
    (
        "require_geo",
        {
            "type": "boolean",
            "title": "Geo-coördinaten vereist",
            "description": "Sla zenders zonder geo-coördinaten over.",
            "default": False,
        },
    ),
]

BEHAVIOUR = [
    (
        "prune_missing",
        {
            "type": "boolean",
            "title": "Niet-passende zenders verwijderen",
            "description": "Verwijdert zenders die niet meer in de gefilterde set zitten. De plugin beheert de hele radiolijst.",
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
            group("Synchronisatie", [control("sync_cron"), control("dry_run")]),
            group("Naamgeving", [control("name_template", {"multi": True})]),
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
                "Vereisten",
                [
                    control("require_countrycode"),
                    control("require_homepage"),
                    control("require_geo"),
                ],
            ),
            group("Gedrag", [control("prune_missing")]),
        ],
    }

    return {
        "name": "Navidrome Radio Sync Plugin",
        "author": "Ruud Deenen",
        "version": "0.2.0",
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
