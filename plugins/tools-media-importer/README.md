# Media Importer

Saves web pages and online videos as vault link notes (`type: link`), extracting readable article text, captions, and local video copies for offline playback.

<!-- more -->

## Features

- **Import from URL**: Captures web articles into structured `article` text versions and video captions into time-aligned `transcript` versions.
- **Offline video playback**: Downloads video streams into `.numen/md.numen.tools.media.importer/<hash>.<ext>` using `yt-dlp` and `ffmpeg` for local playback through `numen.v1.MediaService`.
- **Settings**:
  - **yt-dlp path**: Optional path to the `yt-dlp` executable (resolved from `PATH` when empty).
  - **ffmpeg path**: Optional path to the `ffmpeg` executable (resolved from `PATH` when empty).
