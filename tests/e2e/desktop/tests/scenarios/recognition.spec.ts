import { test, expect } from '../support/test.js'
import { getPaletteField, openVaultFile, openWorkspace } from '../support/workspace.js'

test.describe('A scanned document', () => {
  test('is offered the shared command of creating text once', async ({ page }) => {
    await openWorkspace(page)
    await openVaultFile(page, 'assets', 'regenerative-braking.pdf')

    await page.keyboard.press('Control+p')
    await getPaletteField(page).fill('Create text')

    await expect(page.getByRole('option', { name: /^Create text$/ })).toHaveCount(1)
  })
})
