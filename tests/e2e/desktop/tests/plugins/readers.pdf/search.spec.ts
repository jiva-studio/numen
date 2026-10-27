import { test, expect } from '../../support/test.js'
import { openWorkspace, waitForPassage } from '../../support/workspace.js'

test.describe('PDF Document Reader Search', () => {
  test.describe.configure({ timeout: 120_000 })

  test('finds a passage of the text layer and lights it on its page', async ({ page }) => {
    await openWorkspace(page)

    const passage = await waitForPassage(page, 'shipping manifest', 'assets/flywheel.pdf')
    await passage.click()

    const panel = page.getByRole('tabpanel', { name: /flywheel/i })
    await expect(panel).toBeVisible()
    await expect(
      panel.locator('.reader__page[data-page="0"] .reader__highlight').first(),
    ).toBeVisible()
  })
})
