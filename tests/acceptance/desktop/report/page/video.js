// The video of a scenario: the player, the steps on its timeline, the speed and the pause at each
// step, and the keys. report.js, which the page loads after it, defines the helpers it calls and
// calls createVideo.
/* global createElement, createIcon, formatOffset, writeSetting */
/* exported createVideo */
const playback = window.playback
const SPEED_KEY = 'acceptance-report.speed'
const SPEEDS = [0.25, 0.5, 1]
let currentVideo = null

const readSetting = (key) => {
  try {
    return localStorage.getItem(key)
  } catch {
    return null
  }
}

const readSpeed = () => {
  const saved = Number(readSetting(SPEED_KEY))
  return SPEEDS.includes(saved) ? saved : 1
}

const createSpeedButtons = () =>
  createElement(
    'div',
    { class: 'speed', role: 'group', 'aria-label': 'Playback speed' },
    SPEEDS.map((speed) =>
      createElement('button', { type: 'button', 'data-speed': String(speed) }, [`${speed}×`]),
    ),
  )

const createHint = () =>
  createElement('span', { class: 'hint' }, [
    createElement('kbd', {}, ['Space']),
    ' play · ',
    createElement('kbd', {}, ['←']),
    ' ',
    createElement('kbd', {}, ['→']),
    ' steps',
  ])

const createVideoSection = () => {
  const parts = {
    player: createElement('video', { class: 'player', preload: 'auto', playsinline: '' }),
    playButton: createElement('button', { class: 'icon-button', type: 'button' }),
    time: createElement('span', { class: 'clock' }, ['0:00.0 / 0:00.0']),
    timeline: createElement('div', { class: 'timeline', role: 'group', 'aria-label': 'Steps' }),
    tooltip: createElement('div', { class: 'timeline-tooltip', role: 'tooltip', hidden: '' }),
    speed: createSpeedButtons(),
    pauseToggle: createElement('input', { type: 'checkbox' }),
    fullscreenButton: createElement('button', { class: 'icon-button', type: 'button' }),
  }
  parts.card = createElement('div', { class: 'player-card' }, [
    createElement('div', { class: 'player-frame' }, [parts.player]),
    createElement('div', { class: 'player-bar' }, [
      createElement('div', { class: 'timeline-wrap' }, [parts.timeline, parts.tooltip]),
      createElement('div', { class: 'toolbar' }, [
        parts.playButton,
        parts.time,
        parts.speed,
        createElement('label', { class: 'check' }, [parts.pauseToggle, 'Pause at each step']),
        createHint(),
        parts.fullscreenButton,
      ]),
    ]),
  ])
  parts.section = createElement('section', { class: 'video', 'aria-label': 'Video' }, [parts.card])
  return parts
}

const setIconButton = (button, icon, label) => {
  button.replaceChildren(createIcon(icon))
  button.setAttribute('aria-label', label)
  button.title = label
}

const writePlayButton = (video) =>
  video.player.paused
    ? setIconButton(video.playButton, 'play', 'Play')
    : setIconButton(video.playButton, 'pause', 'Pause')

const writeFullscreenButton = (video) =>
  document.fullscreenElement === video.card
    ? setIconButton(video.fullscreenButton, 'shrink', 'Exit full screen')
    : setIconButton(video.fullscreenButton, 'expand', 'Full screen')

const toggleFullscreen = (video) => {
  const request =
    document.fullscreenElement === video.card
      ? document.exitFullscreen()
      : video.card.requestFullscreen()
  // The browser may refuse full screen; the button stays as it is.
  request.catch(() => writeFullscreenButton(video))
}

const setSpeed = (video, speed) => {
  video.player.defaultPlaybackRate = speed
  video.player.playbackRate = speed
  for (const button of video.speed.children)
    button.setAttribute('aria-pressed', String(Number(button.dataset.speed) === speed))
}

const getStarts = (video) => video.scenario.steps.map((step) => step.start)

// Steps that start together share one moment of the video; the one clicked stays current there.
const findCurrentStep = (video, time) => {
  const starts = getStarts(video)
  const index = playback.findStepAt(starts, time)
  const picked = video.pickedIndex
  return picked !== undefined && starts[picked] === starts[index] ? picked : index
}

const writePosition = (video) => {
  const time = video.player.currentTime
  const phases = playback.getStepPhases(video.stepItems.length, findCurrentStep(video, time))
  video.stepItems.forEach((item, index) => {
    for (const phase of ['done', 'current', 'ahead'])
      item.classList.toggle(phase, phases[index] === phase)
  })
  for (const segment of video.timeline.querySelectorAll('.segment')) {
    const phase = phases[Number(segment.dataset.index)]
    for (const name of ['done', 'current', 'ahead']) segment.classList.toggle(name, phase === name)
  }
  video.time.textContent = `${formatOffset(time)} / ${formatOffset(video.length)}`
  const playhead = video.timeline.querySelector('.playhead')
  if (playhead && video.length > 0)
    playhead.style.left = `${Math.min((time * 100) / video.length, 100)}%`
}

const createSegment = (video, segment) => {
  const step = video.scenario.steps[segment.index]
  return createElement('button', {
    class: `segment ${step.status}`,
    type: 'button',
    'data-index': String(segment.index),
    'aria-label': `${step.keyword} ${step.text}`,
    style: `left: ${segment.left}%; width: ${segment.width}%`,
  })
}

const writeTimeline = (video) => {
  video.length = playback.getVideoLength(video.player.duration, video.scenario.steps)
  const segments = playback.getSegments(getStarts(video), video.length)
  video.timeline.replaceChildren(
    ...segments.map((segment) => createSegment(video, segment)),
    createElement('div', { class: 'playhead' }),
  )
  writePosition(video)
}

const showTooltip = (video, segment) => {
  const step = video.scenario.steps[Number(segment.dataset.index)]
  video.tooltip.replaceChildren(
    createElement('span', { class: 'keyword' }, [step.keyword]),
    step.text,
  )
  video.tooltip.hidden = false
  const wrap = video.tooltip.parentElement.getBoundingClientRect()
  const box = segment.getBoundingClientRect()
  const half = video.tooltip.offsetWidth / 2
  const center = box.left + box.width / 2 - wrap.left
  video.tooltip.style.left = `${Math.min(Math.max(center, half), wrap.width - half)}px`
}

// play() fails when the browser blocks playback or the video cannot load; the page shows the position.
const playVideo = (video) => {
  video.player.play().catch(() => writePosition(video))
}

const seekToStep = (video, index) => {
  const start = getStarts(video)[index]
  if (start === null || start === undefined) return
  video.pickedIndex = index
  video.player.pause()
  seekTo(video, start)
}

const seekTo = (video, time) => {
  video.player.currentTime = time
  writePosition(video)
}

const togglePlay = (video) => {
  if (video.player.paused) playVideo(video)
  else video.player.pause()
}

const goToNextStep = (video) => {
  if (video.isPausingAtSteps && video.player.paused) {
    playVideo(video)
    return
  }
  const next = playback.findNextStart(getStarts(video), video.player.currentTime)
  if (next === null) return
  // Pausing at each step, the video stops at the step it skipped to.
  if (video.isPausingAtSteps) video.player.pause()
  seekTo(video, next)
}

const goToPreviousStep = (video) =>
  seekTo(video, playback.findPreviousStart(getStarts(video), video.player.currentTime))

const setPausePoint = (video) => {
  video.pauseAt = video.isPausingAtSteps
    ? playback.findNextStart(getStarts(video), video.player.currentTime)
    : null
}

// Follows the video frame by frame while it plays: `timeupdate` comes four times a second, too
// seldom to stop at a step's start or to move the playhead smoothly.
const followPlayback = (video) => {
  const { player } = video
  if (player.paused || !player.isConnected) return
  if (video.pauseAt !== null && player.currentTime >= video.pauseAt) {
    player.pause()
    player.currentTime = video.pauseAt
  }
  writePosition(video)
  video.frame = requestAnimationFrame(() => followPlayback(video))
}

/**
 * Shows a frame from the middle of the video before it first plays. A second element takes the
 * frame: seeking the player itself would drop its poster.
 */
const writePoster = (video) => {
  const probe = createElement('video', { muted: '', preload: 'auto', src: video.source })
  probe.muted = true
  probe.addEventListener(
    'loadedmetadata',
    () => {
      probe.currentTime = playback.getVideoLength(probe.duration, video.scenario.steps) / 2
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

const linkTimeline = (video) => {
  video.timeline.addEventListener('click', (event) => {
    const segment = event.target.closest('.segment')
    if (segment) seekToStep(video, Number(segment.dataset.index))
  })
  const show = (event) => {
    const segment = event.target.closest('.segment')
    if (segment) showTooltip(video, segment)
  }
  const hide = () => {
    video.tooltip.hidden = true
  }
  video.timeline.addEventListener('pointerover', show)
  video.timeline.addEventListener('focusin', show)
  video.timeline.addEventListener('pointerleave', hide)
  video.timeline.addEventListener('focusout', hide)
}

const linkControls = (video) => {
  video.speed.addEventListener('click', (event) => {
    const button = event.target.closest('button')
    if (!button) return
    const speed = Number(button.dataset.speed)
    setSpeed(video, speed)
    writeSetting(SPEED_KEY, String(speed))
  })
  video.pauseToggle.addEventListener('change', () => {
    video.isPausingAtSteps = video.pauseToggle.checked
    setPausePoint(video)
  })
  video.playButton.addEventListener('click', () => togglePlay(video))
  video.player.addEventListener('click', () => togglePlay(video))
  video.fullscreenButton.addEventListener('click', () => toggleFullscreen(video))
  video.card.addEventListener('fullscreenchange', () => writeFullscreenButton(video))
  writePlayButton(video)
  writeFullscreenButton(video)
}

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

const linkVideo = (video) => {
  const { player } = video
  player.addEventListener('loadedmetadata', () => writeTimeline(video))
  player.addEventListener('durationchange', () => writeTimeline(video))
  player.addEventListener('timeupdate', () => writePosition(video))
  player.addEventListener('seeked', () => {
    setPausePoint(video)
    writePosition(video)
  })
  player.addEventListener('pause', () => writePlayButton(video))
  player.addEventListener('play', () => {
    writePlayButton(video)
    setPausePoint(video)
    cancelAnimationFrame(video.frame)
    video.frame = requestAnimationFrame(() => followPlayback(video))
  })
  video.stepList.addEventListener('click', (event) => {
    const item = event.target.closest('.step')
    if (item && !event.target.closest('.error, .doc-string, .screenshot'))
      seekToStep(video, Number(item.dataset.index))
  })
  linkTimeline(video)
  linkControls(video)
  setSpeed(video, readSpeed())
}

const KEY_ACTIONS = { ' ': togglePlay, ArrowRight: goToNextStep, ArrowLeft: goToPreviousStep }

const isTyping = (target) =>
  target instanceof HTMLTextAreaElement ||
  (target instanceof HTMLInputElement && target.type !== 'checkbox')

const handleKey = (event) => {
  const action = KEY_ACTIONS[event.key]
  if (!action || !currentVideo?.player.isConnected) return
  if (event.altKey || event.ctrlKey || event.metaKey || isTyping(event.target)) return
  // Ahead of a focused button, which would take Space as its own click.
  event.preventDefault()
  event.stopPropagation()
  action(currentVideo)
}

/** The video section of a scenario, linked to the list of its steps. */
const createVideo = (scenario, stepList) => {
  const parts = createVideoSection()
  if (currentVideo?.source.startsWith('blob:')) URL.revokeObjectURL(currentVideo.source)
  currentVideo = {
    ...parts,
    scenario,
    stepList,
    stepItems: [...stepList.children],
    length: 0,
    pauseAt: null,
    isPausingAtSteps: false,
    frame: 0,
    source: '',
  }
  linkVideo(currentVideo)
  loadVideo(currentVideo)
  return parts.section
}

window.addEventListener('keydown', handleKey, { capture: true })
