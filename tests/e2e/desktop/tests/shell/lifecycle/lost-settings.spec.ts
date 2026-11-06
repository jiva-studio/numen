import { test, expect } from '../../support/test.js'
import { expectWorkspaceDrawn } from '../../support/workspace.js'

/** How long the answer to the settings request is held: longer than the window waits for it. */
const SETTINGS_HOLD_MS = 15_000

/**
 * How soon the workspace shell is on the screen while the settings have not answered: the window waits
 * up to 3 s for the plugins of its interface, then draws.
 */
const SHELL_WITHIN_MS = 5_000

test.describe('Start-up while the settings request never answers', () => {
  test('draws the workspace once the wait for the interface plugins passes, then says what did not load and loads on retry', async ({
    page,
  }) => {
    let isHolding = true
    await page.route('**/numen.v1.SettingsService/GetSettings', async (route) => {
      if (isHolding) await new Promise((resolve) => setTimeout(resolve, SETTINGS_HOLD_MS))
      await route.continue().catch(() => undefined)
    })

    await page.goto('/')
    await expect(page.locator('section[data-workspace-pane]').first()).toBeVisible({
      timeout: SHELL_WITHIN_MS,
    })

    const failure = page.getByRole('alert').filter({ hasText: 'Could not load the workspace' })
    await expect(failure).toContainText('The settings request did not answer in 10 s', {
      timeout: 12_000,
    })

    isHolding = false
    await failure.getByRole('button', { name: 'Retry' }).click()

    await expect(failure).toBeHidden()
    await expectWorkspaceDrawn(page)
  })
})
