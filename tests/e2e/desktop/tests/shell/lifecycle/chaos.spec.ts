import { test, expect } from '../../support/test.js'
import { openFromFiles } from '../../support/workspace.js'

test.describe('Chaos, Fault Tolerance and Stress Scenarios', () => {
  test('renders high-volume document smoothly without interface freeze', async ({ page }) => {
    await openFromFiles(page, ['notes', 'notes/large.md'])

    // Verify large document editor is rendered and contains expected content
    await expect(page.locator('.note').first()).toBeVisible({ timeout: 10_000 })
    const content = page.locator('.note__text .cm-content').first()
    await expect(content).toBeVisible({ timeout: 10_000 })
    await expect(content).toContainText('Large Document')
    await expect(content).toContainText('Paragraph line 1')
  })
})
