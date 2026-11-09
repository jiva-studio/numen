/**
 * The highlight a tab shows over the text a search found. A book lights the words; a PDF is
 * drawn as a picture and its backend turns text into rectangles only, so its page shows that a
 * highlight lies on it but not what text is under it.
 */
import { expect, type Locator, type Page } from 'playwright/test'
import { getEscaped, getStem } from './rules.js'

const readLitText = (): string => {
  const ranges = CSS.highlights.get('numen-book')
  return ranges ? [...ranges].map((range) => range.toString()).join(' ') : ''
}

/** Checks that the tab of the file shows a highlight, and in a book its text. */
export const expectTabHighlight = async (page: Page, file: string, text?: string): Promise<void> => {
  const panel: Locator = page.getByRole('tabpanel', { name: new RegExp(getEscaped(getStem(file)), 'i') })
  await expect(panel).toBeVisible()
  if (file.endsWith('.epub') && text !== undefined) {
    await expect(panel.locator('.book-tab')).toBeVisible()
    await expect.poll(() => page.evaluate(readLitText)).toContain(text)
  } else {
    await expect(panel.locator('.reader__page .reader__highlight').first()).toBeVisible()
  }
}
