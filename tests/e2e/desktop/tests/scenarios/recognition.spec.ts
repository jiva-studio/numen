import { test, expect } from 'playwright/test'
import {
  getPaletteField,
  openVaultFile,
  openWorkspace,
  runCommand,
  waitForPassage,
} from '../support/workspace.js'

/** Words of the text the mock recogniser of `-debug-mock-ml` reads on every page. */
const MOCK_RECOGNISED_WORDS = 'recovers kinetic energy'

test.describe('A scanned document', () => {
  test.describe.configure({ timeout: 180_000 })

  test('is recognised, found through the search, and opened with the place highlighted', async ({
    page,
  }) => {
    await openWorkspace(page)
    const tab = await openVaultFile(page, 'assets', 'regenerative-braking.pdf')
    await expect(tab.getByRole('img', { name: 'Page 1' })).toBeVisible()

    await runCommand(page, 'Recognise', /^Recognise/)

    const passage = await waitForPassage(
      page,
      MOCK_RECOGNISED_WORDS,
      'assets/regenerative-braking.pdf',
    )
    await expect(passage).toContainText(/kinetic energy/i)
    await passage.click()

    const reader = page.getByRole('tabpanel', { name: /regenerative-braking/i })
    await expect(reader.getByRole('img', { name: 'Page 1' })).toBeVisible()
    await expect(reader.locator('.reader__highlight').first()).toBeVisible()
  })

  test('is offered the recognise command of the OCR plugin once', async ({ page }) => {
    await openWorkspace(page)
    await openVaultFile(page, 'assets', 'regenerative-braking.pdf')

    await page.keyboard.press('Control+p')
    await getPaletteField(page).fill('Recognise')

    await expect(page.getByRole('option', { name: /Document OCR/ })).toHaveCount(1)
  })
})
