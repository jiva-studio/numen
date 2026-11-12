/**
 * The highlight a tab shows over the text a search found. A book lights the words; a PDF is
 * drawn as a picture and its backend turns text into rectangles only, so its page shows that a
 * highlight lies on it but not what text is under it.
 */
import { expect } from 'playwright/test'
import { TabPanel, TabPanels } from './components/TabPanel.js'
import type { Host } from './components/host.js'

const readLitText = (): string => {
  const ranges = CSS.highlights.get('numen-book')
  return ranges ? [...ranges].map((range) => range.toString()).join(' ') : ''
}

/** Checks that the tab of the file shows a highlight, and in a book its text. */
export const expectTabHighlight = async (host: Host, file: string, text?: string): Promise<void> => {
  const { page } = host
  const panel = new TabPanel(new TabPanels(host).ofFile(file))
  await expect(panel.panel).toBeVisible()
  if (file.endsWith('.epub') && text !== undefined) {
    await expect(panel.book()).toBeVisible()
    await expect.poll(() => page.evaluate(readLitText)).toContain(text)
  } else {
    await expect(panel.pageHighlight()).toBeVisible()
  }
}
