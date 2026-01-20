# Developer Manual

This document explains how the YouTube download + upload stub works, how to run it locally, and where to make changes.

## System Overview
- **Entry point:** `main.go` parses CLI flags, validates prerequisites (`yt-dlp`, optional `ffmpeg`), prepares the sqlite metadata store, and wires together the controller, downloader, and uploader components.
- **Controller (`controller.go`):** Orchestrates a sync. For channel syncs it enumerates videos, skips those already recorded in sqlite, downloads fresh items via the downloader, sends resulting files to the uploader, and marks them uploaded.
- **Downloader (`downloader.go`):** Thin wrapper around `yt-dlp`. It can list video IDs from a channel and download an individual video while printing the paths of the produced files. Retries SABR/DASH failures with dynamic MPD options.
- **Uploader stub (`main.go`):** Implements the `uploader` interface by logging every file path. Replace this with a real implementation per platform.
- **Persistence (`store.go`):** `SQLiteStore` ensures the schema and records `(video_id, channel_id, uploaded_at)` tuples so the controller can skip work that already finished.
- **HTTP mode (`http.go`):** When `--http-addr` is set, an HTTP server exposes `POST /sync` to trigger channel syncs asynchronously.

```
CLI/HTTP → Controller → Downloader ──yt-dlp──→ files → Uploader
                              ↘ SQLite store (tracks uploaded IDs)
```

## Directory Layout
| Path | Purpose |
| ---- | ------- |
| `main.go` | Flag parsing, dependency wiring, CLI/HTTP mode selection |
| `controller.go` | Sync orchestration logic |
| `downloader.go` | `yt-dlp` wrappers for listing/downloading |
| `store.go` | Sqlite persistence helpers |
| `http.go` | Minimal HTTP server for remote triggering |
| `main_test.go` | Unit tests for config helpers and format/runtime selection |
| `downloads/` | Default output directory (created automatically) |
| `metadata.db` | Sqlite database (auto-generated) |

## Local Setup
1. Install Go 1.22+.
2. Install `yt-dlp` (`brew install yt-dlp` on macOS) and ensure it is in `PATH`.
3. Install `ffmpeg` if you want merged mp4 outputs; without it you get single-stream fallback.
4. Clone the repo and run `go test ./...` to confirm everything compiles.
5. Optional: build once with `go build ./...` to warm the module cache, especially before using Docker builds.

### Running from the CLI
```bash
go run . --video-id dQw4w9WgXcQ --platform bilibili
go run . --channel-id UC_x5XG1OV2P6uZZ5FSM9Ttw --limit 3 --sleep-seconds 5
```
Key flags:
- `--channel-id` / `--video-id`: Provide one (unless running HTTP server mode). Channel IDs can also be full URLs.
- `--platform`: Lowercase platform label passed to the uploader stub (`bilibili` or `tiktok` by default).
- `--output`: Output directory (defaults to `downloads`); auto-created.
- `--db-path`: Location of the sqlite database.
- `--js-runtime`: Influences the `--js-runtimes` flag passed to `yt-dlp`. `"auto"` selects `node` or `deno` that actually exists in `PATH`.
- `--format`: Custom `yt-dlp` format string. `"auto"` prefers mp4 when `ffmpeg` exists; otherwise uses a single-stream fallback.
- `--sleep-seconds`: Adds `--sleep-interval` flags to avoid rate limiting.

### HTTP Controller Mode
```bash
go run . --http-addr :8080 --output downloads
```
`POST /sync` with a JSON body: `{"channel_id":"UC123","limit":3}`. The handler times out after 30 minutes. Responses include `{considered, skipped, downloaded, uploaded}` counts plus `error` when a step fails.

## Persistence Model
- Sqlite lives at `--db-path` (default `metadata.db`) and contains a single `uploads` table. `video_id` is the primary key.
- `Controller.SyncChannel` checks `Store.IsUploaded` before downloading new files.
- After a successful upload, `Store.MarkUploaded` upserts the video ID and timestamp. If the sync ran for a single video (no channel context) the channel is stored as `"unknown"`.

## Downloader Details
- Lists channel IDs via `yt-dlp --flat-playlist --print id`.
- Downloads use `--print after_postprocess:filepath` (when `ffmpeg` exists) or `after_move` otherwise so the controller knows the produced filenames.
- SABR/DASH fallbacks: if stderr mentions `"SABR streaming"`, `HTTP Error 403`, etc., the downloader retries with `--allow-dynamic-mpd --concurrent-fragments 1`.
- JS runtimes: `resolveDesiredJSRuntime` inspects `yt-dlp --help` output once to ensure the binary supports `--js-runtimes`. If not, `"auto"` silently disables the flag, but explicit values fail fast.

## Uploader Integration Points
To support a real platform:
1. Implement the `uploader` interface in a new file (e.g., `bilibili_uploader.go`).
2. Inject the new struct in `main.go` in place of `dummyUploader` by branching on `cfg.platform`.
3. Ensure uploads return an error when the remote API fails so the controller stops before marking a video as uploaded.

For multi-platform handling consider moving uploader creation to a `newUploader(platform string) (uploader, error)` helper.

## Testing
- `go test ./...` hits `main_test.go`, which validates flag parsing, URL helpers, JS runtime selection, and format fallback logic.
- Downloader/controller/store pieces hit real binaries and sqlite, so integration tests would need to stub or mock `yt-dlp`. Preferred approach: extract interfaces for running commands and inject fakes.

## Docker & Distribution
`Dockerfile` builds the CLI inside a container. Mount the host downloads directory so files persist:
```bash
docker build -t yt-transfer .
docker run --rm -v "$PWD/downloads:/app/downloads" yt-transfer --channel-id UC... --limit 3
```

## Common Development Tasks
- **Clear sqlite state:** Delete `metadata.db` (or use a different `--db-path`) to force re-download/upload of all videos.
- **Change defaults:** Update the `config` struct defaults in `parseFlagsFrom`.
- **Add logging:** All components use the standard library `log` package with timestamps disabled for cleaner CLI output.
- **Extend HTTP API:** Add more handlers in `http.go` and update the mux before the call to `http.ListenAndServe`.

Questions or edge cases not covered here? Check the source files listed above—they mirror the structure of this manual and are intentionally small.
