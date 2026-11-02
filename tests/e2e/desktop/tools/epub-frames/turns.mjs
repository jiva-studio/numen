// The page turns of a recording: forward one at a time, forward in a rapid run, back one at a time,
// back in a rapid run. Each key is posted with the clock in milliseconds before it is sent.
{
const post = (message) => window.webkit.messageHandlers.log.postMessage(String(message))
const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms))
const getFocused = () => {
  let element = document.activeElement
  while (element?.shadowRoot?.activeElement) element = element.shadowRoot.activeElement
  return element ?? document.body
}
const press = (key) => {
  post(`PRESS ${Date.now()} ${key}`)
  for (const type of ['keydown', 'keyup']) {
    getFocused().dispatchEvent(
      new KeyboardEvent(type, { key, code: key, bubbles: true, composed: true, cancelable: true }),
    )
  }
}
const repeat = async (key, times, gap) => {
  for (let time = 0; time < times; time += 1) {
    press(key)
    await sleep(gap)
  }
}
const play = async () => {
  await repeat('ArrowRight', 6, 1000)
  await repeat('PageDown', 8, 90)
  await sleep(1000)
  await repeat('ArrowLeft', 2, 1000)
  await repeat('PageUp', 8, 90)
  await sleep(1200)
  post('TURNS-DONE')
}
void play()
}
