/* global createElement */
/* exported createCursor, writeCursor, linkCursor */
const CURSOR_NS = 'http://www.w3.org/2000/svg'
const CURSOR_PATH = 'M5 2.5v16.2l4.1-3.9 2.7 6.3 2.9-1.2-2.7-6.2h5.7z'

const createPointer = () => {
  const svg = document.createElementNS(CURSOR_NS, 'svg')
  svg.setAttribute('class', 'cursor-pointer')
  svg.setAttribute('viewBox', '0 0 24 24')
  const path = document.createElementNS(CURSOR_NS, 'path')
  path.setAttribute('d', CURSOR_PATH)
  svg.append(path)
  return svg
}

const createCursor = () => {
  const pointer = createPointer()
  const ripple = createElement('span', { class: 'cursor-ripple' })
  const press = createElement('span', { class: 'cursor-press' })
  const highlightLayer = createElement('div', { class: 'input-highlights' })
  const badge = createElement('div', { class: 'key-badge' })
  const element = createElement('div', { class: 'cursor-layer', 'aria-hidden': 'true' }, [
    highlightLayer,
    ripple,
    press,
    pointer,
    badge,
  ])
  return { element, pointer, ripple, press, highlights: highlightLayer, badge }
}

const getContentBox = (player) => {
  const { videoWidth, videoHeight, offsetWidth, offsetHeight, offsetLeft, offsetTop } = player
  if (!(videoWidth > 0 && videoHeight > 0)) return null
  const scale = Math.min(offsetWidth / videoWidth, offsetHeight / videoHeight)
  const width = videoWidth * scale
  const height = videoHeight * scale
  return {
    left: offsetLeft + (offsetWidth - width) / 2,
    top: offsetTop + (offsetHeight - height) / 2,
    width,
    height,
  }
}

const placeLayer = (layer, box) => {
  layer.style.left = `${box.left}px`
  layer.style.top = `${box.top}px`
  layer.style.width = `${box.width}px`
  layer.style.height = `${box.height}px`
}

const placePoint = (element, x, y) => {
  element.style.left = `${x * 100}%`
  element.style.top = `${y * 100}%`
}

const writeRipple = (ripple, state) => {
  ripple.hidden = state === null
  if (state === null) return
  placePoint(ripple, state.x, state.y)
  ripple.style.setProperty('--progress', String(state.progress))
}

const writePointer = (cursor, state) => {
  cursor.pointer.style.display = state ? '' : 'none'
  cursor.press.hidden = !state?.isPressed
  writeRipple(cursor.ripple, state?.ripple ?? null)
  if (!state) return
  placePoint(cursor.pointer, state.x, state.y)
  placePoint(cursor.press, state.x, state.y)
}

const writeParts = (element, parts) => {
  const text = parts.join('\n')
  if (element.dataset.text === text) return
  element.dataset.text = text
  element.replaceChildren(
    ...parts.map((part) => createElement('span', { class: 'key-part' }, [part])),
  )
}

const createHighlight = () => {
  const label = createElement('span', { class: 'input-label' })
  return createElement('span', { class: 'input-highlight' }, [label])
}

const getActionLabel = (inputs, ring) => {
  const upTo = inputs.slice(0, ring.index + 1)
  return window.playback.getBadgeAt(upTo, inputs[ring.index].t)?.parts.at(-1) ?? ''
}

const writeHighlight = (highlight, ring, label) => {
  placePoint(highlight, ring.x, ring.y)
  const room = 1 - ring.x
  highlight.style.width = ring.w < room - 0.005 ? `${ring.w * 100}%` : `calc(${room * 100}% - 6px)`
  highlight.style.height = `${ring.h * 100}%`
  highlight.style.opacity = String(ring.opacity)
  highlight.classList.toggle('is-near-bottom', ring.y + ring.h > 0.9)
  highlight.classList.toggle('has-label', Boolean(label))
  const tab = highlight.firstElementChild
  tab.hidden = !label
  if (label) writeParts(tab, [label])
}

const writeHighlights = (container, rings, inputs) => {
  while (container.children.length < rings.length) container.append(createHighlight())
  ;[...container.children].forEach((highlight, index) => {
    const ring = rings[index]
    highlight.hidden = !ring
    if (ring) writeHighlight(highlight, ring, getActionLabel(inputs, ring))
  })
}

const writeBadge = (badge, state) => {
  badge.hidden = state === null
  if (state === null) return
  badge.style.opacity = String(state.opacity)
  writeParts(badge, state.parts)
}

const writeCursor = (video) => {
  const { cursor, player } = video
  const box = getContentBox(player)
  cursor.element.hidden = !box
  if (!box) return
  const inputs = video.scenario.inputs ?? []
  const time = player.currentTime
  placeLayer(cursor.element, box)
  writePointer(cursor, window.playback.getCursorAt(inputs, time))
  const rings = window.playback.getHighlightsAt(inputs, time)
  writeHighlights(cursor.highlights, rings, inputs)
  writeBadge(cursor.badge, rings.length > 0 ? null : window.playback.getBadgeAt(inputs, time))
}

const linkCursor = (video) => {
  video.player.after(video.cursor.element)
  const observer = new ResizeObserver(() => writeCursor(video))
  observer.observe(video.player)
  video.player.addEventListener('loadedmetadata', () => writeCursor(video))
}
