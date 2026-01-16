# YouTube Downloader + Upload Stub (Go)

Minimal CLI to download from a YouTube channel or single video ID, then pass results to a stub uploader for China platforms.

## Requirements
- `yt-dlp` in `PATH`
- `ffmpeg` recommended (improves format handling)

## Usage
```bash
go run . --video-id dQw4w9WgXcQ --platform bilibili
go run . --channel-id UC_x5XG1OV2P6uZZ5FSM9Ttw --limit 3 --sleep-seconds 5
```

Options:
- `--channel-id` YouTube channel ID or URL
- `--video-id` YouTube video ID or URL
- `--platform` `bilibili` or `tiktok`
- `--output` output directory (default: `downloads`)
- `--limit` max videos for channel downloads (default: 5)
- `--sleep-seconds` sleep between downloads to reduce rate (default: 5)

## Docker
```bash
docker build -t yt-transfer .
docker run --rm -v "$PWD/downloads:/app/downloads" yt-transfer \
  --channel-id UC_x5XG1OV2P6uZZ5FSM9Ttw --limit 3 --sleep-seconds 5
```

## Notes
- Uploads are stubbed; the app just logs the files to upload.
- For channel downloads, the tool limits to the newest `--limit` videos.
