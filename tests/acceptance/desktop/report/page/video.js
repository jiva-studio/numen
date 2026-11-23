/* global createElement, formatOffset, loadVideo, createControl, createSpeed, createHelp, createClock, createPauseToggle, setControl, showControls, setSpeed, closeSpeedMenu, linkSpeedMenu, linkIdle, readSpeed */
/* exported createVideo, getOpenTime */
const playback = window.playback
let currentVideo = null

const createControls = (parts) =>
  createElement('div', { class: 'controls' }, [
    createElement('div', { class: 'timeline-wrap' }, [parts.timeline, parts.tooltip]),
    createElement('div', { class: 'toolbar' }, [
      parts.playButton,
      parts.clock.element,
      createElement('span', { class: 'spacer' }),
      parts.pauseToggle,
      parts.speed.element,
      createHelp(),
      parts.fullscreenButton,
    ]),
  ])

/* global createCursor, writeCursor, linkCursor */
const createVideoSection = () => {
  const parts = {
    player: createElement('video', { class: 'player', preload: 'auto', playsinline: '' }),
    playButton: createControl('play-button', 'Play'),
    clock: createClock(),
    timeline: createElement('div', { class: 'timeline', role: 'group', 'aria-label': 'Steps' }),
    tooltip: createElement('div', { class: 'timeline-tooltip', role: 'tooltip', hidden: '' }),
    speed: createSpeed(),
    pauseToggle: createPauseToggle(),
    fullscreenButton: createControl('fullscreen-button', 'Full screen'),
  }
  parts.card = createElement('div', { class: 'player-card' }, [parts.player, createControls(parts)])
  parts.section = createElement('section', { class: 'video', 'aria-label': 'Video' }, [parts.card])
  return parts
}

const writePlayButton = (video) =>
  video.player.paused
    ? setControl(video.playButton, 'play', 'Play')
    : setControl(video.playButton, 'pause', 'Pause')

const writeFullscreenButton = (video) =>
  document.fullscreenElement === video.card
    ? setControl(video.fullscreenButton, 'minimize', 'Exit full screen')
    : setControl(video.fullscreenButton, 'maximize', 'Full screen')

const toggleFullscreen = (video) => {
  const request =
    document.fullscreenElement === video.card
      ? document.exitFullscreen()
      : video.card.requestFullscreen()
  request.catch(() => writeFullscreenButton(video))
}

const getStarts = (video) => video.scenario.steps.map((step) => step.start)

const findFailedIndex = (video) =>
  video.scenario.steps.findIndex((step) => step.status === 'failed' && step.start !== null)

const findActionIndex = (video) =>
  getStarts(video).findIndex((start, index) => index >= video.setupLength && start !== null)

const findOpenIndex = (video) => {
  const failed = findFailedIndex(video)
  return failed === -1 ? findActionIndex(video) : failed
}

const getOpenTime = (video) => {
  const failed = findFailedIndex(video)
  if (failed !== -1) return getStarts(video)[failed]
  return playback.findActionStart(getStarts(video), video.setupLength) ?? 0
}

const findCurrentStep = (video, time) => {
  const starts = getStarts(video)
  const index = playback.findStepAt(starts, time)
  const picked = video.pickedIndex
  return picked !== undefined && starts[picked] === starts[index] ? picked : index
}

const getFill = (segment, percent) => {
  const left = Number(segment.dataset.left)
  const width = Number(segment.dataset.width)
  return Math.min(Math.max((percent - left) / width, 0), 1)
}

const writeSegments = (video, phases, percent) => {
  for (const segment of video.timeline.querySelectorAll('.segment')) {
    const phase = phases[Number(segment.dataset.index)]
    for (const name of ['done', 'current', 'ahead']) segment.classList.toggle(name, phase === name)
    segment.style.setProperty('--fill', String(getFill(segment, percent)))
  }
}

const writeSteps = (video, phases) =>
  video.stepItems.forEach((item, index) => {
    for (const phase of ['done', 'current', 'ahead'])
      item.classList.toggle(phase, phases[index] === phase)
  })

const writePosition = (video) => {
  const time = video.player.currentTime
  const percent = video.length > 0 ? Math.min((time * 100) / video.length, 100) : 0
  const phases = playback.getStepPhases(video.stepItems.length, findCurrentStep(video, time))
  writeSteps(video, phases)
  writeSegments(video, phases, percent)
  video.clock.current.textContent = formatOffset(time)
  video.clock.total.textContent = formatOffset(video.length)
  const playhead = video.timeline.querySelector('.playhead')
  if (playhead) playhead.style.left = `${percent}%`
  writeCursor(video)
}

const createSegment = (video, segment) => {
  const step = video.scenario.steps[segment.index]
  return createElement(
    'button',
    {
      class: `segment ${step.status}`,
      type: 'button',
      'data-index': String(segment.index),
      'data-left': String(segment.left),
      'data-width': String(segment.width),
      'aria-label': `${step.keyword} ${step.text}`,
      style: `left: ${segment.left}%; width: ${segment.width}%`,
    },
    [createElement('span', { class: 'segment-fill' })],
  )
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

const getTooltipText = (video, segment) => {
  const index = Number(segment.dataset.index)
  const step = video.scenario.steps[index]
  return [
    createElement('span', { class: 'tooltip-number' }, [String(index + 1)]),
    createElement('span', { class: 'keyword' }, [step.keyword]),
    step.text,
  ]
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

const startVideo = (video) => {
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
  if (video.player.paused) startVideo(video)
  else video.player.pause()
}

const goToNextStep = (video) => {
  if (video.isPausingAtSteps && video.player.paused) {
    startVideo(video)
    return
  }
  const next = playback.findNextStart(getStarts(video), video.player.currentTime)
  if (next === null) return
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

const togglePausing = (video) => {
  video.isPausingAtSteps = !video.isPausingAtSteps
  video.pauseToggle.setAttribute('aria-pressed', String(video.isPausingAtSteps))
  setPausePoint(video)
}

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
  video.pauseToggle.addEventListener('click', () => togglePausing(video))
  video.playButton.addEventListener('click', () => togglePlay(video))
  video.player.addEventListener('click', () => togglePlay(video))
  video.fullscreenButton.addEventListener('click', () => toggleFullscreen(video))
  video.card.addEventListener('fullscreenchange', () => writeFullscreenButton(video))
  linkSpeedMenu(video)
  linkIdle(video)
  writePlayButton(video)
  writeFullscreenButton(video)
}

const openVideo = (video) => {
  const index = findOpenIndex(video)
  video.pickedIndex = index === -1 ? undefined : index
  seekTo(video, getOpenTime(video))
}

const handleStepClick = (video, event) => {
  if (event.target.closest('.error, .doc-string, .screenshot, .table-scroll')) return
  const item = event.target.closest('.step')
  if (item) seekToStep(video, Number(item.dataset.index))
}

const handlePlay = (video) => {
  writePlayButton(video)
  setPausePoint(video)
  showControls(video)
  cancelAnimationFrame(video.frame)
  video.frame = requestAnimationFrame(() => followPlayback(video))
}

const handlePause = (video) => {
  writePlayButton(video)
  showControls(video)
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
  player.addEventListener('pause', () => handlePause(video))
  player.addEventListener('play', () => handlePlay(video))
  video.stepList.addEventListener('click', (event) => handleStepClick(video, event))
  linkTimeline(video)
  linkControls(video)
  linkCursor(video)
  setSpeed(video, readSpeed())
}

const KEY_ACTIONS = { ' ': togglePlay, ArrowRight: goToNextStep, ArrowLeft: goToPreviousStep }

const isTyping = (target) =>
  target instanceof HTMLTextAreaElement || target instanceof HTMLInputElement

const isInMenu = (target) => target instanceof Element && target.closest('.menu') !== null

const isOnButton = (event) =>
  event.key === ' ' && event.target instanceof Element && event.target.closest('button') !== null

const isForPage = (event) =>
  event.altKey ||
  event.ctrlKey ||
  event.metaKey ||
  isTyping(event.target) ||
  isInMenu(event.target) ||
  isOnButton(event)

const handleKey = (event) => {
  const action = KEY_ACTIONS[event.key]
  if (!action || !currentVideo?.player.isConnected || isForPage(event)) return
  event.preventDefault()
  event.stopPropagation()
  action(currentVideo)
  showControls(currentVideo)
}

const createVideo = (scenario, stepList) => {
  const parts = createVideoSection()
  if (currentVideo?.source.startsWith('blob:')) URL.revokeObjectURL(currentVideo.source)
  if (currentVideo) closeSpeedMenu(currentVideo)
  currentVideo = {
    ...parts,
    cursor: createCursor(),
    scenario,
    stepList,
    stepItems: [...stepList.querySelectorAll('.step')],
    setupLength: playback.getSetupLength(scenario.steps),
    length: 0,
    pauseAt: null,
    isPausingAtSteps: false,
    frame: 0,
    idleTimer: 0,
    source: '',
  }
  linkVideo(currentVideo)
  loadVideo(currentVideo)
  return parts.section
}

window.addEventListener('keydown', handleKey, { capture: true })
