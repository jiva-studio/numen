/**
 * The highlight a tab shows over the text a search found. A book lights the words; a PDF is
 * drawn as a picture, so its page can only show that a highlight lies on it.
 */
import { expect, type Locator, type Page } from 'playwright/test'
import { getEscaped, getStem } from './rules.js'

const readLitText = (): string => {
  const ranges = CSS.highlights.get('numen-book')
  return ranges ? [...ranges].map((range) => range.toString()).join(' ') : ''
}

/** Checks that the tab of the file shows a highlight, and in a book that the highlight holds the text. */
export const expectTabHighlight = async (page: Page, file: string, text: string): Promise<void> => {
  const panel: Locator = page.getByRole('tabpanel', { name: new RegExp(getEscaped(getStem(file)), 'i') })
  await expect(panel).toBeVisible()
  if (file.endsWith('.epub')) {
    await expect(panel.locator('.book-tab')).toBeVisible()
    await expect.poll(() => page.evaluate(readLitText)).toContain(text)
  } else {
    await expect(panel.locator('.reader__page .reader__highlight').first()).toBeVisible()
  }
}
