import { test, expect } from '../../support/test.js'
import { startOwnServer, type OwnServer } from '../../support/server.js'
import { observeFor } from '../../support/browser.js'
import { expectWorkspaceDrawn, getPaletteField, USABLE_WITHIN_MS } from '../../support/workspace.js'

/** How long the embeddings plugin takes to open its model: far longer than the run. */
const MODEL_OPEN_MS = 120_000

test.describe('Start-up while the embeddings model is still opening', () => {
  let server: OwnServer

  test.beforeAll(async () => {
    server = await startOwnServer([], { env: { NUMEN_MOCK_OPEN_HOLD_MS: String(MODEL_OPEN_MS) } })
  })

  test.afterAll(async () => {
    await server.stop()
  })

  test('draws the workspace, shows the indexing as running and answers a search by words', async ({
    page,
  }) => {
    const errors: string[] = []
    page.on('pageerror', (error) => errors.push(error.message))
    await page.goto(server.url)
    await expectWorkspaceDrawn(page)

    const within = { timeout: USABLE_WITHIN_MS }
    await expect(page.locator('[data-tasks-status] button')).toContainText(/\d+ of \d+/, within)

    // The embedding pass is now waiting for the model, which stays closed for the rest of the run.
    // A window that loads now asks the index and the search while the plugin holds the model.
    await observeFor(page, 2_000)
    await page.reload()
    await expectWorkspaceDrawn(page)
    await expect(page.locator('button:has(.index-stats__figure)')).toBeVisible(within)
    await page.keyboard.press('Control+k')
    await getPaletteField(page).fill('Entropy')
    await expect(page.getByRole('option').first()).toBeVisible(within)
    expect(errors).toStrictEqual([])
  })
})
