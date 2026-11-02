import { test, expect } from '../../support/test.js'
import { openWorkspace } from '../../support/workspace.js'
import { getRow, showInFiles } from '../../support/layout.js'

declare global {
  interface Window {
    /** The texts the page wrote to the clipboard since it loaded. */
    copiedTexts?: string[]
  }
}

test.describe('Copy path', () => {
  test('puts the vault path of the row on the clipboard', async ({ page }) => {
    await page.addInitScript(() => {
      window.copiedTexts = []
      Object.defineProperty(navigator, 'clipboard', {
        configurable: true,
        value: {
          writeText: (text: string) => {
            window.copiedTexts?.push(text)
            return Promise.resolve()
          },
        },
      })
    })
    await openWorkspace(page)
    await showInFiles(page, ['notes'])

    await getRow(page, 'notes/Entropy.md').click({ button: 'right' })
    await page.getByRole('menuitem', { name: 'Copy path', exact: true }).click()

    await expect.poll(() => page.evaluate(() => window.copiedTexts)).toEqual(['notes/Entropy.md'])
  })
})
