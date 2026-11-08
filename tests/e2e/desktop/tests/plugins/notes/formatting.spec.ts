import { test, expect } from '../../support/test.js'
import { openFromFiles } from '../../support/workspace.js'

test.describe('Note editor live formatting and styling', () => {

  test('renders bulleted list with bullet marker widget in editor', async ({ page }) => {
    await openFromFiles(page, ['notes', 'notes/plain.md'])

    const cmContent = page.locator('.note .cm-content').first()
    await expect(cmContent).toBeVisible({ timeout: 10_000 })

    // Type bullet list item
    await cmContent.click()
    await page.keyboard.press('End')
    await page.keyboard.type('\n- Fundamental principle of entropy\n')

    // Verify list item text is present
    await expect(cmContent).toContainText('Fundamental principle of entropy')

    // Verify bullet widget or bullet mark
    await expect(page.locator('.cm-bullet').first()).toBeVisible({ timeout: 5_000 })
  })
})
