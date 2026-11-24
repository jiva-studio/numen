/**
 * Saves the text as a file the person keeps, with the save dialog of the browser. It answers true once the
 * download started: the browser does not say whether the person kept the file.
 */
export function saveFile(name: string, content: string): Promise<boolean> {
  const address = URL.createObjectURL(new Blob([content], { type: 'application/json' }))
  const link = document.createElement('a')
  link.href = address
  link.download = name
  document.body.append(link)
  link.click()
  link.remove()
  setTimeout(() => URL.revokeObjectURL(address), 0)
  return Promise.resolve(true)
}
