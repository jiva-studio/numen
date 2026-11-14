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
  const element = createElement('div', { class: 'cursor-layer', 'aria-hidden': 'true' }, [
    ripple,
    pointer,
  ])
  return { element, pointer, ripple }
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

const writeCursor = (video) => {
  const { cursor, player } = video
  const box = getContentBox(player)
  const state = box && window.playback.getCursorAt(video.scenario.clicks ?? [], player.currentTime)
  cursor.element.hidden = !state
  if (!state) return
  placeLayer(cursor.element, box)
  placePoint(cursor.pointer, state.x, state.y)
  writeRipple(cursor.ripple, state.ripple)
}

const linkCursor = (video) => {
  video.player.after(video.cursor.element)
  const observer = new ResizeObserver(() => writeCursor(video))
  observer.observe(video.player)
  video.player.addEventListener('loadedmetadata', () => writeCursor(video))
}
