// Turns back to the first page of the book, so every recording starts from the same page.
{
  const post = (message) => window.webkit.messageHandlers.log.postMessage(String(message))
  const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms))
  const getFocused = () => {
    let element = document.activeElement
    while (element?.shadowRoot?.activeElement) element = element.shadowRoot.activeElement
    return element ?? document.body
  }
  const rewind = async () => {
    for (let time = 0; time < 60; time += 1) {
      for (const type of ['keydown', 'keyup']) {
        getFocused().dispatchEvent(
          new KeyboardEvent(type, { key: 'PageUp', code: 'PageUp', bubbles: true, composed: true, cancelable: true }),
        )
      }
      await sleep(150)
    }
    await sleep(1500)
    post('RESET-DONE')
  }
  void rewind()
}
