# Media Downloader Plugin

Media and web page downloader with transcript extraction via `yt-dlp`.

## Features

- Downloads video and audio streams from supported URLs via `yt-dlp` and `ffmpeg`.
- Serves `numen.v1.DownloadService`.
- Configurable `yt_dlp` and `ffmpeg` binary paths via plugin settings.
- Activates on `onCommand:downloadMedia` and `onCommand:downloader.open`.
