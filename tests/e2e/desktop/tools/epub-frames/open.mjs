// Opens a book from the Files pane of the window and posts READY with the reading area once the
// reader has drawn it. The name of the book is set by the caller as window.BOOK_PATH before this runs.
{
const post = (message) => window.webkit.messageHandlers.log.postMessage(String(message))
const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms))
const findAll = (selector) => {
  const found = []
  const visit = (root) => {
    found.push(...root.querySelectorAll(selector))
    for (const element of root.querySelectorAll('*')) if (element.shadowRoot) visit(element.shadowRoot)
  }
  visit(document)
  return found
}
// The reader is drawn in a plugin's open shadow root, and a book left open earlier is hidden there.
window.findShown = (selector) =>
  findAll(selector).find((element) => element.getBoundingClientRect().width > 0)
const waitFor = async (find) => {
  for (let tries = 0; tries < 300; tries += 1) {
    const found = find()
    if (found) return found
    await sleep(100)
  }
  throw new Error(`not found: ${find}`)
}
const press = (element, key) => {
  element.focus()
  element.dispatchEvent(new KeyboardEvent('keydown', { key, code: key, bubbles: true }))
}
const openBook = async () => {
  const path = window.BOOK_PATH ?? 'assets/ledger.epub'
  const folder = path.split('/')[0]
  await waitFor(() => document.querySelector('[role=tab]'))
  const files = [...document.querySelectorAll('[role=tab]')].find((tab) =>
    /^Files/.test(tab.textContent.trim()),
  )
  files.click()
  const folderRow = await waitFor(() => document.querySelector(`[data-tree-row="${folder}"]`))
  if (!document.querySelector(`[data-tree-row="${path}"]`)) press(folderRow, 'Enter')
  press(await waitFor(() => document.querySelector(`[data-tree-row="${path}"]`)), 'Enter')
  await waitFor(() => window.findShown('.book__paper:not(.book__paper--unset)'))
  await sleep(1500)
  const area = window.findShown('.book__area')
  area.tabIndex = -1
  area.focus()
  const box = area.getBoundingClientRect()
  post(`AREA ${Math.round(box.x)},${Math.round(box.y)},${Math.round(box.width)},${Math.round(box.height)}`)
  post('READY')
}
void openBook().catch((error) => post(`ERR ${error}`))
}
