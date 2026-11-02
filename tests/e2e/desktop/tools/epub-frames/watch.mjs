// Posts the footer counter and the translate of the paper whenever either changes, and every key
// pressed, each with the clock in milliseconds, so the frames of a recording can be set against them.
{
const post = (message) => window.webkit.messageHandlers.log.postMessage(String(message))
const counter = window.findShown('.book__count')
const paper = window.findShown('.book__paper')
let last = ''
const sample = () => {
  const now = `${counter.textContent} ${paper.style.translate}`
  if (now !== last) {
    last = now
    post(`OBS ${Date.now()} ${now}`)
  }
  requestAnimationFrame(sample)
}
document.addEventListener('keydown', (event) => post(`KEY ${Date.now()} ${event.key}`), true)
sample()
}
