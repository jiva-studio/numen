import { test, expect } from 'playwright/test'
import { openVaultFile, openWorkspace, runCommand, waitForPassage } from '../support/workspace.js'

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
      'recovers kinetic energy',
      'assets/regenerative-braking.pdf',
    )
    await expect(passage).toContainText(/kinetic energy/i)
    await passage.click()

    const reader = page.getByRole('tabpanel', { name: /regenerative-braking/i })
    await expect(reader.getByRole('img', { name: 'Page 1' })).toBeVisible()
    await expect(reader.locator('.reader__highlight').first()).toBeVisible()
  })
})
