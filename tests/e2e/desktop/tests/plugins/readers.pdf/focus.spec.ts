import { test, expect } from '../../support/test.js'
import { openWorkspace, openVaultFile } from '../../support/workspace.js'

test.describe('The pages of a PDF in focus', () => {
  test('draw no ring inside the reader when the keyboard has focused them', async ({ page }) => {
    await openWorkspace(page)
    await openVaultFile(page, 'assets', 'pages.pdf')
    const viewport = page.locator('.reader__viewport:visible')
    await expect(page.locator('.reader__page[data-page="0"]')).toBeAttached({ timeout: 20_000 })

    // The keyboard is the last input, so a focus moves in with :focus-visible.
    await page.keyboard.press('ArrowRight')
    await viewport.focus()

    expect(await viewport.evaluate((one) => one.matches(':focus-visible'))).toBe(true)
    const look = await viewport.evaluate((one) => {
      const style = getComputedStyle(one)
      return { shadow: style.boxShadow, outline: style.outlineStyle }
    })
    expect(look).toEqual({ shadow: 'none', outline: 'none' })
  })
})
