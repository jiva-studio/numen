# Media Importer Plugin

Media and web page downloader with transcript extraction via `yt-dlp`.

## Features

- Downloads video and audio streams from supported URLs via `yt-dlp` and `ffmpeg`.
- Serves `numen.v1.DownloadService`.
- Configurable `yt_dlp` and `ffmpeg` binary paths via plugin settings.
- Declares `yt-dlp` and `ffmpeg` as `binary` dependencies (system-first, no download). The host injects the resolved path as `NUMEN_DEPENDENCY_YT_DLP` / `NUMEN_DEPENDENCY_FFMPEG`. Program precedence: plugin setting path, then injected variable, then `PATH`.
- Activates on `onCommand:downloadMedia` and `onCommand:downloader.open`.
