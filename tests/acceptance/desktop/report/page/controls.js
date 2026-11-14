/* global createElement, createIcon, writeSetting */
/* exported createControl, createSpeed, createHelp, createClock, createPauseToggle, setControl, showControls, setSpeed, closeSpeedMenu, linkSpeedMenu, linkIdle, readSpeed */
const SPEED_KEY = 'acceptance-report.speed'
const SPEEDS = [0.25, 0.5, 1]
const IDLE_MS = 2000
const SHORTCUTS = [
  ['Space', 'Play or pause'],
  ['←', 'Previous step'],
  ['→', 'Next step'],
]

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

const formatSpeed = (speed) => `${speed}×`

const createControl = (className, tooltip) =>
  createElement('button', {
    class: `control ${className}`,
    type: 'button',
    'aria-label': tooltip,
    'data-tooltip': tooltip,
  })

const setControl = (button, icon, label) => {
  button.replaceChildren(createIcon(icon))
  button.setAttribute('aria-label', label)
  button.dataset.tooltip = label
}

const createSpeedMenu = () =>
  createElement(
    'div',
    { class: 'menu', role: 'menu', 'aria-label': 'Playback speed', hidden: '' },
    SPEEDS.map((speed) =>
      createElement(
        'button',
        { type: 'button', role: 'menuitemradio', 'data-speed': String(speed) },
        [createIcon('check'), formatSpeed(speed)],
      ),
    ),
  )

const createSpeed = () => {
  const button = createControl('speed-button', 'Playback speed')
  button.setAttribute('aria-haspopup', 'menu')
  button.setAttribute('aria-expanded', 'false')
  const menu = createSpeedMenu()
  return { button, menu, element: createElement('div', { class: 'speed' }, [button, menu]) }
}

const createShortcut = ([key, action]) =>
  createElement('li', {}, [createElement('span', {}, [action]), createElement('kbd', {}, [key])])

const createHelp = () =>
  createElement('div', { class: 'help' }, [
    createElement(
      'button',
      { class: 'control text-control', type: 'button', 'aria-label': 'Shortcuts' },
      ['?'],
    ),
    createElement('div', { class: 'shortcuts', role: 'tooltip' }, [
      createElement('p', {}, ['Shortcuts']),
      createElement('ul', {}, SHORTCUTS.map(createShortcut)),
    ]),
  ])

const createClock = () => {
  const current = createElement('span', {}, ['0:00.0'])
  const total = createElement('span', { class: 'clock-total' }, ['0:00.0'])
  return {
    current,
    total,
    element: createElement('span', { class: 'clock' }, [current, ' / ', total]),
  }
}

const createPauseToggle = () => {
  const button = createControl('pause-toggle', 'Pause at each step')
  button.setAttribute('aria-pressed', 'false')
  button.append(createIcon('stops'))
  return button
}

const isIdleAllowed = (video) =>
  !video.player.paused && video.speed.menu.hidden && !video.card.querySelector(':focus-visible')

const hideControls = (video) => {
  if (isIdleAllowed(video)) video.card.classList.add('is-idle')
}

const showControls = (video) => {
  video.card.classList.remove('is-idle')
  clearTimeout(video.idleTimer)
  if (!video.player.paused) video.idleTimer = setTimeout(() => hideControls(video), IDLE_MS)
}

const setSpeed = (video, speed) => {
  video.player.defaultPlaybackRate = speed
  video.player.playbackRate = speed
  video.speed.button.textContent = formatSpeed(speed)
  for (const item of video.speed.menu.children)
    item.setAttribute('aria-checked', String(Number(item.dataset.speed) === speed))
}

const closeSpeedMenu = (video) => {
  video.speed.menu.hidden = true
  video.speed.button.setAttribute('aria-expanded', 'false')
  document.removeEventListener('pointerdown', video.closeOnOutside, true)
}

const openSpeedMenu = (video) => {
  video.speed.menu.hidden = false
  video.speed.button.setAttribute('aria-expanded', 'true')
  video.speed.menu.querySelector('[aria-checked="true"]')?.focus()
  document.addEventListener('pointerdown', video.closeOnOutside, true)
}

const toggleSpeedMenu = (video) =>
  video.speed.menu.hidden ? openSpeedMenu(video) : closeSpeedMenu(video)

const pickSpeed = (video, item) => {
  const speed = Number(item.dataset.speed)
  setSpeed(video, speed)
  writeSetting(SPEED_KEY, String(speed))
  closeSpeedMenu(video)
  video.speed.button.focus()
}

const closeOnEscape = (video, event) => {
  if (event.key !== 'Escape') return
  closeSpeedMenu(video)
  video.speed.button.focus()
}

const linkSpeedMenu = (video) => {
  video.closeOnOutside = (event) => {
    if (!event.target.closest('.speed')) closeSpeedMenu(video)
  }
  video.speed.button.addEventListener('click', () => toggleSpeedMenu(video))
  video.speed.menu.addEventListener('click', (event) => {
    const item = event.target.closest('[data-speed]')
    if (item) pickSpeed(video, item)
  })
  video.speed.menu.addEventListener('keydown', (event) => closeOnEscape(video, event))
}

const linkIdle = (video) => {
  video.card.addEventListener('pointermove', () => showControls(video))
  video.card.addEventListener('pointerleave', () => hideControls(video))
  video.card.addEventListener('focusin', () => showControls(video))
}
