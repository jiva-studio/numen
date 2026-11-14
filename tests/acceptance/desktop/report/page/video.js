// The video of a scenario: the player, the steps on its timeline, the speed and the pause at each
// step, and the keys. report.js, which the page loads after it, defines the helpers it calls and
// calls createVideo.
/* global createElement, createIcon, formatOffset, loadVideo, writeSetting */
/* exported createVideo, getOpenTime */
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

// The moments the step keys and the pause at each step stop at: the setup is one stop, at 0.
const getStops = (video) =>
  getStarts(video).map((start, index) => (index < video.setupLength ? null : start))

const isSetupFailed = (video) =>
  video.scenario.steps.slice(0, video.setupLength).some((step) => step.status === 'failed')

/** Where the video opens: at the first action, past the start of the app; at 0 if the setup failed. */
const getOpenTime = (video) =>
  isSetupFailed(video) ? 0 : (playback.findActionStart(getStarts(video), video.setupLength) ?? 0)

// Steps that start together share one moment of the video; the one clicked stays current there.
const findCurrentStep = (video, time) => {
  const starts = getStarts(video)
  const index = playback.findStepAt(starts, time)
  const picked = video.pickedIndex
  return picked !== undefined && starts[picked] === starts[index] ? picked : index
}

const writeSetupPhase = (video, currentIndex) => {
  const phase = currentIndex < video.setupLength ? 'current' : 'done'
  for (const element of [video.setupRow, video.timeline.querySelector('.segment.setup')]) {
    if (!element) continue
    for (const name of ['done', 'current']) element.classList.toggle(name, phase === name)
  }
}

const writePosition = (video) => {
  const time = video.player.currentTime
  const currentIndex = findCurrentStep(video, time)
  const phases = playback.getStepPhases(video.stepItems.length, currentIndex)
  video.stepItems.forEach((item, index) => {
    for (const phase of ['done', 'current', 'ahead'])
      item.classList.toggle(phase, phases[index] === phase)
  })
  if (video.setupLength > 0) writeSetupPhase(video, currentIndex)
  for (const segment of video.timeline.querySelectorAll('.segment:not(.setup)')) {
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

const getSetupStatus = (video) =>
  playback.getWorstStatus(video.scenario.steps.slice(0, video.setupLength).map((s) => s.status))

const getSetupLabel = (video) =>
  `Setup · ${video.setupLength} step${video.setupLength === 1 ? '' : 's'}`

/** The setup takes the timeline from 0 to the first action, or to the end when no action ran. */
const createSetupSegment = (video) => {
  const end = playback.findActionStart(getStarts(video), video.setupLength) ?? video.length
  const width = Math.min((end * 100) / video.length, 100)
  return createElement('button', {
    class: `segment setup ${getSetupStatus(video)}`,
    type: 'button',
    'aria-label': getSetupLabel(video),
    style: `left: 0%; width: ${width}%`,
  })
}

const writeTimeline = (video) => {
  video.length = playback.getVideoLength(video.player.duration, video.scenario.steps)
  const segments = playback.getSegments(getStops(video), video.length)
  const setup = video.setupLength > 0 && video.length > 0 ? [createSetupSegment(video)] : []
  video.timeline.replaceChildren(
    ...setup,
    ...segments.map((segment) => createSegment(video, segment)),
    createElement('div', { class: 'playhead' }),
  )
  writePosition(video)
}

const getTooltipText = (video, segment) => {
  if (segment.classList.contains('setup')) return [getSetupLabel(video)]
  const step = video.scenario.steps[Number(segment.dataset.index)]
  return [createElement('span', { class: 'keyword' }, [step.keyword]), step.text]
}

const showTooltip = (video, segment) => {
  video.tooltip.replaceChildren(...getTooltipText(video, segment))
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

const seekToSetup = (video) => {
  video.pickedIndex = undefined
  video.player.pause()
  seekTo(video, 0)
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
  const next = playback.findNextStart(getStops(video), video.player.currentTime)
  if (next === null) return
  // Pausing at each step, the video stops at the step it skipped to.
  if (video.isPausingAtSteps) video.player.pause()
  seekTo(video, next)
}

const goToPreviousStep = (video) =>
  seekTo(video, playback.findPreviousStart(getStops(video), video.player.currentTime))

const setPausePoint = (video) => {
  video.pauseAt = video.isPausingAtSteps
    ? playback.findNextStart(getStops(video), video.player.currentTime)
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

const linkTimeline = (video) => {
  video.timeline.addEventListener('click', (event) => {
    const segment = event.target.closest('.segment')
    if (segment?.classList.contains('setup')) seekToSetup(video)
    else if (segment) seekToStep(video, Number(segment.dataset.index))
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

// The step the video opens at stays current though steps after it may share its start.
const openVideo = (video) => {
  const time = getOpenTime(video)
  video.pickedIndex = time > 0 ? video.setupLength : undefined
  seekTo(video, time)
}

const handleStepClick = (video, event) => {
  if (event.target.closest('.setup-toggle, .error, .doc-string, .screenshot')) return
  if (event.target.closest('.setup-row')) {
    seekToSetup(video)
    return
  }
  const item = event.target.closest('.step')
  if (item) seekToStep(video, Number(item.dataset.index))
}

const linkVideo = (video) => {
  const { player } = video
  player.addEventListener('loadedmetadata', () => writeTimeline(video))
  player.addEventListener('loadedmetadata', () => openVideo(video), { once: true })
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
  video.stepList.addEventListener('click', (event) => handleStepClick(video, event))
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
    stepItems: [...stepList.querySelectorAll('.step')],
    setupRow: stepList.querySelector('.setup-row'),
    setupLength: playback.getSetupLength(scenario.steps),
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
