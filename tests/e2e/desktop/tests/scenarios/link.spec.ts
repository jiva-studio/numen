import { test, expect, type Page } from '../support/test.js'
import { getPaletteField, openWorkspace, runCommand, waitForPassage } from '../support/workspace.js'

/** Served by the proxy `scripts/start-server.mjs` puts in front of every plain-http fetch. */
const ARTICLE_ADDRESS = 'http://tidal-mills.numen.test/history/rance'

/**
 * A video site's address. Under `NUMEN_MOCK_ML` the media importer contacts
 * video sites through its stand-in downloader, which publishes captions and
 * copies a three-second clip, offline.
 */
const VIDEO_ADDRESS = 'https://www.youtube.com/watch?v=tidalmill01'

const ARTICLE_TEXT = /A tidal mill stored seawater behind a sluice gate/

/** A second video, so its link note is not the one the other scenario makes. */
const SECOND_VIDEO_ADDRESS = 'https://www.youtube.com/watch?v=tidalmill02'

/** The link note the importer names after the second video's address, at the root of the vault. */
const LINK_NOTE = 'https---www.youtube.com-watch-v=tidalmill02.md'

const readOpenPaths = async (page: Page): Promise<string[]> => {
  const answer = await page.request.post('/numen.v1.WorkspaceService/ReadOpenTabs', {
    data: {},
    headers: { 'Content-Type': 'application/json' },
  })
  const open = (await answer.json()) as { tabs?: { path?: string }[] }
  return (open.tabs ?? []).map((tab) => tab.path ?? '')
}

const importAddress = async (page: Page, address: string): Promise<void> => {
  await runCommand(page, 'Import from URL', /^Import from URL/)
  await getPaletteField(page).fill(address)
  await page.keyboard.press('Enter')
}

test.describe('A link', () => {
  test('to a page shows the article fetched from it, and the search finds it', async ({ page }) => {
    await openWorkspace(page)
    await importAddress(page, ARTICLE_ADDRESS)

    await expect(page.getByRole('tabpanel').getByText(ARTICLE_TEXT)).toBeVisible({
      timeout: 30_000,
    })

    const passage = await waitForPassage(page, 'sluice gate', 'tidal-mills.numen.test')
    await expect(passage).toContainText(/sluice gate/i)
  })

  test('keeps its article deleted once the person deletes it', async ({ page }) => {
    await openWorkspace(page)
    await importAddress(page, `${ARTICLE_ADDRESS}?part=2`)
    const tab = page.getByRole('tabpanel').filter({ hasText: 'tidal-mills.numen.test' })
    await expect(tab.getByText(ARTICLE_TEXT)).toBeVisible({ timeout: 30_000 })

    await runCommand(page, 'Delete text', /^Delete text$/)
    await expect(tab.getByText('The text of this link was deleted.')).toBeVisible({
      timeout: 30_000,
    })

    await page.reload()
    await expect(tab.getByText('The text of this link was deleted.')).toBeVisible({
      timeout: 30_000,
    })
    await expect(tab.getByText(ARTICLE_TEXT)).toHaveCount(0)
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

  test('to a video opens from the files tab with its copy and its captions', async ({ page }) => {
    test.setTimeout(120_000)
    await openWorkspace(page)
    await importAddress(page, SECOND_VIDEO_ADDRESS)
    const caption = 'The mill pond fills behind the gate.'
    const imported = page.getByRole('tabpanel').filter({ hasText: caption })
    await expect(imported.getByRole('button', { name: /^Play/ })).toBeVisible({ timeout: 30_000 })

    await openWorkspace(page)
    await page.getByRole('tab', { name: /^Files/ }).click()
    await page.getByRole('treeitem', { name: LINK_NOTE, exact: true }).press('Enter')

    const fromTree = page.getByRole('tabpanel').filter({ hasText: caption })
    await expect(fromTree.getByRole('button', { name: /^Play/ })).toBeVisible({ timeout: 30_000 })
    await expect.poll(() => readOpenPaths(page)).toContain(LINK_NOTE)

    await page.reload()

    const restored = page.getByRole('tabpanel').filter({ hasText: caption })
    await expect(restored.getByRole('button', { name: /^Play/ })).toBeVisible({ timeout: 30_000 })
  })
})
