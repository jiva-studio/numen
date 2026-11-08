import { test, expect, type Page } from '../support/test.js'
import {
  closeTab,
  getPaletteField,
  openWorkspace,
  runCommand,
  runCommandOverItem,
  waitForPassage,
} from '../support/workspace.js'
import { listFilePaths } from '../support/vault.js'
import { deleteVaultFiles } from '../support/media.js'

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

const importAddress = async (page: Page, address: string, tab: RegExp): Promise<void> => {
  await runCommand(page, 'Import link', /^Import link/)
  await getPaletteField(page).fill(address)
  await page.keyboard.press('Enter')
  await expect(page.getByRole('tab', { name: tab, selected: true })).toBeVisible({
    timeout: 30_000,
  })
}

const downloadText = (page: Page): Promise<void> =>
  runCommand(page, 'Download captions', /^Download captions$/)

const downloadVideo = (page: Page): Promise<void> =>
  runCommandOverItem(page, 'Download media', /^Download media$/, /^Video$/)

test.describe('A link', () => {
  let before: string[] = []

  test.beforeEach(async ({ request }) => {
    before = await listFilePaths(request)
  })

  // The link notes a test made go, so the next run imports the same addresses again.
  test.afterEach(async ({ request }) => {
    deleteVaultFiles(...(await listFilePaths(request)).filter((one) => !before.includes(one)))
  })

  test('to a page shows the article fetched from it, and the search finds it', async ({ page }) => {
    await openWorkspace(page)
    await importAddress(page, ARTICLE_ADDRESS, /rance(?!-part)/)
    await downloadText(page)

    await expect(page.getByRole('tabpanel').getByText(ARTICLE_TEXT)).toBeVisible({
      timeout: 30_000,
    })

    const passage = await waitForPassage(page, 'sluice gate', 'tidal-mills.numen.test')
    await expect(passage).toContainText(/sluice gate/i)
  })

  test('imported again opens the note the address already has', async ({ page, request }) => {
    await openWorkspace(page)
    await importAddress(page, `${ARTICLE_ADDRESS}?part=3`, /rance-part=3/)
    const made = (await listFilePaths(request)).filter((one) => !before.includes(one))
    await closeTab(page, /^Close .*rance-part=3/)
    await expect(page.getByRole('tab', { name: /rance-part=3/ })).toHaveCount(0)

    await importAddress(page, `${ARTICLE_ADDRESS}?part=3`, /rance-part=3/)

    expect((await listFilePaths(request)).filter((one) => !before.includes(one))).toEqual(made)
  })

  test('keeps its article deleted once the person deletes it', async ({ page }) => {
    await openWorkspace(page)
    await importAddress(page, `${ARTICLE_ADDRESS}?part=2`, /rance-part=2/)
    await downloadText(page)
    const tab = page.getByRole('tabpanel', { name: /rance-part=2/ })
    await expect(tab.getByText(ARTICLE_TEXT)).toBeVisible({ timeout: 30_000 })

    await runCommandOverItem(page, 'Delete text', /^Delete text$/, /\(current\)$/)
    await expect(tab.getByText(ARTICLE_TEXT)).toHaveCount(0, { timeout: 30_000 })

    await page.reload()
    await expect(tab).toBeVisible({ timeout: 30_000 })
    await expect(tab.getByText(ARTICLE_TEXT)).toHaveCount(0)
  })

  test('to a video keeps a copy that plays', async ({ page }) => {
    await openWorkspace(page)
    await importAddress(page, VIDEO_ADDRESS, /tidalmill01/)
    await downloadText(page)
    await downloadVideo(page)

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
    await importAddress(page, SECOND_VIDEO_ADDRESS, /tidalmill02/)
    await downloadText(page)
    await downloadVideo(page)
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
