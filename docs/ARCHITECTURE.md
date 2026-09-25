# Architecture

```
Navidrome (WASM sandbox)
  └── navidrome-radio-plugin.ndp
        main.go                 Lifecycle / Scheduler / TaskWorker entry points
        internal/
          settings     config -> Settings (defaults, list normalization)
          radiobrowser  radio-browser client (mirrors, retries, paging) + Station
          template     {field} / {field?:default} name rendering
          filter       country/tag/language/codec/bitrate/broken filters
          syncer       plan / index / apply / finalize (added in step 3)
          subsonic     Subsonic radio API wrapper (added in step 3)
```

## Host services used

| Service | Use |
|---------|-----|
| `Config` | Read settings. |
| `HTTP` | Fetch radio-browser pages. |
| `Scheduler` | Recurring sync (cron) + one-time initial run. |
| `Task` | Batched, retryable background work. |
| `KVStore` | Sync cursor + station ID mapping. |
| `SubsonicAPI` | `get/create/update/deleteInternetRadioStation`. |
| `Users` | Pick an admin user for the Subsonic calls. |

## Constraints (from the Navidrome 0.64.1 plugin runtime)

| Constraint | Value | Consequence |
|---|---|---|
| Plugin call timeout | 30 s | every task must stay short |
| Task payload | 1 MB | small batches |
| Outbound HTTP response | 10 MB | paginate radio-browser (page_size ≤ ~5000) |
| `SubsonicAPICall` | in-process | full radio list ≈ 11 MB for ~55k stations |
| Radio ID | random (`id.NewRandom()`) | a freshly created ID is only known after a new full list fetch |
| create/update/delete | admin only | plugin must call as an admin user |
| Task queue `DelayMs` | built-in rate limiter | throttling knob for the initial import |

## Sync design

Because the Subsonic API has no bulk create, the full list is reconciled:

1. `plan` – fetch radio-browser pages, filter + render, persist the desired set
   and a cursor in KVStore.
2. `index` – one `getInternetRadioStations` call builds `stream_url → id`.
3. `apply` – per batch: match on stream URL, update when changed, otherwise
   create (a duplicate-name error means it already exists).
4. `finalize` – re-index to learn newly created IDs and prune stations that no
   longer match (when `prune_missing`).

All phases are resumable through the KVStore cursor and idempotent.

## Why a plugin (and not direct SQLite writes)

The earlier `navidrome-radio-sync.py` wrote directly into the `radio` table.
A plugin is sandboxed and has no database access, so it uses the supported
Subsonic API instead. The trade-off is that the initial import is one API call
per station; adoption of already-present stations keeps this near zero.

## Testing

`internal/...` is free of PDK imports, so `go test ./internal/...` runs on any
platform. HTTP is injected through a `Doer` interface, so the radio-browser
client is tested with a fake.
