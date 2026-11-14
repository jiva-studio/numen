// Loads the video of a scenario into its player and draws its poster. video.js, loaded before it,
// defines getOpenTime; report.js defines createElement.
/* global createElement, getOpenTime */
/* exported loadVideo */

/**
 * The video held in memory: through a server without range requests, such as Python's http.server,
 * the browser cannot seek in it. A page opened from file:// cannot fetch and plays the file itself.
 */
const fetchVideo = async (url) => {
  try {
    const response = await fetch(url)
    return response.ok ? URL.createObjectURL(await response.blob()) : url
  } catch {
    return url
  }
}

const setSource = (video, source) => {
  video.source = source
  video.player.src = source
  writePoster(video)
}

const loadVideo = (video) => {
  const url = video.scenario.video
  fetchVideo(url).then(
    (source) => setSource(video, source),
    () => setSource(video, url),
  )
}

/**
 * Shows the frame the video opens at before it first plays. A second element takes the
 * frame: seeking the player itself would drop its poster.
 */
const writePoster = (video) => {
  const probe = createElement('video', { muted: '', preload: 'auto', src: video.source })
  probe.muted = true
  probe.addEventListener(
    'loadedmetadata',
    () => {
      probe.currentTime = getOpenTime(video)
    },
    { once: true },
  )
  probe.addEventListener('seeked', () => drawPoster(video.player, probe), { once: true })
}

/** The frame as an image, or null: a page opened from file:// may not read a frame back. */
const readFrame = (probe) => {
  const canvas = createElement('canvas')
  canvas.width = probe.videoWidth
  canvas.height = probe.videoHeight
  try {
    canvas.getContext('2d').drawImage(probe, 0, 0)
    return canvas.toDataURL('image/jpeg', 0.85)
  } catch {
    return null
  }
}

const drawPoster = (player, probe) => {
  const frame = probe.videoWidth > 0 ? readFrame(probe) : null
  if (frame) player.poster = frame
  probe.removeAttribute('src')
  probe.load()
}
