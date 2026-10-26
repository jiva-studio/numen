import { test, expect, type Page } from 'playwright/test'
import { getPaletteField, openWorkspace, runCommand, waitForPassage } from '../support/workspace.js'

/** Served by the proxy `scripts/start-server.mjs` puts in front of every plain-http fetch. */
const ARTICLE_ADDRESS = 'http://tidal-mills.numen.test/history/rance'

/**
 * A video site's address. Under `NUMEN_MOCK_ML` the media importer reaches video sites through its
 * mock downloader, which publishes captions and copies a clip of three seconds, offline.
 */
const VIDEO_ADDRESS = 'https://www.youtube.com/watch?v=tidalmill01'

const importAddress = async (page: Page, address: string): Promise<void> => {
  await runCommand(page, 'Import', /^Import/)
  await getPaletteField(page).fill(address)
  await page.keyboard.press('Enter')
}

test.describe('A link', () => {
  test('to a page shows the article fetched from it, and the search finds it', async ({ page }) => {
    await openWorkspace(page)
    await importAddress(page, ARTICLE_ADDRESS)

    await expect(
      page.getByRole('tabpanel').getByText(/A tidal mill stored seawater behind a sluice gate/),
    ).toBeVisible({ timeout: 30_000 })

    const passage = await waitForPassage(page, 'sluice gate', 'tidal-mills.numen.test')
    await expect(passage).toContainText(/sluice gate/i)
  })

  test('to a video keeps a copy that plays', async ({ page }) => {
    await openWorkspace(page)
    await importAddress(page, VIDEO_ADDRESS)

    const tab = page
      .getByRole('tabpanel')
      .filter({ hasText: 'The mill pond fills behind the gate.' })
    const play = tab.getByRole('button', { name: /^Play/ })
    await expect(play).toBeVisible({ timeout: 30_000 })
    await play.click()

    await expect(tab.getByRole('button', { name: /^Pause/ })).toBeVisible()
    await expect(tab).toContainText(/0:0[1-9]/)
  })
})
