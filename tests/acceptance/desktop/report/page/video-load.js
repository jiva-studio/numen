/* global createElement, getOpenTime */
/* exported loadVideo */

const fetchVideo = async (url) => {
  try {
    const response = await fetch(url)
    return response.ok ? URL.createObjectURL(await response.blob()) : url
  } catch {
    return url
  }
}

const setSource = (video, source) => {
  if (!video.player.isConnected) {
    if (source.startsWith('blob:')) URL.revokeObjectURL(source)
    return
  }
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
