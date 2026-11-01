import { test, expect, type Page } from '../../support/test.js'
import { getPaletteField, openWorkspace, runCommand } from '../../support/workspace.js'
import { deleteVaultFiles, listPluginFiles } from '../../support/media.js'
import { listFilePaths } from '../../support/vault.js'

/** The plugin that imports links, and keeps their copies in its folder of the vault. */
const IMPORTER = 'md.numen.tools.media.importer'

/** A caption the mock downloader of `-debug-mock-ml` publishes for every video. */
const CAPTION = 'The mill pond fills behind the gate.'

/** An address the proxy of `scripts/start-server.mjs` answers 404 for. */
const MISSING_PAGE_ADDRESS = 'http://unreachable.numen.test/tide-tables'

/** The copies of videos the importer keeps in its folder. */
const listCopies = (): string[] => listPluginFiles(IMPORTER).filter((name) => name.endsWith('.wav'))

/** The link note the importer names after a video's address, at the root of the vault. */
const getLinkNote = (id: string): string => `https---www.youtube.com-watch-v=${id}.md`

const importAddress = async (page: Page, address: string): Promise<void> => {
  await runCommand(page, 'Import from URL', /^Import from URL/)
  await getPaletteField(page).fill(address)
  await page.keyboard.press('Enter')
}

test.describe('a link to a video', () => {
  test.describe.configure({ timeout: 120_000 })

  const made: string[] = []

  test.beforeEach(async ({ page }) => {
    // A link without a copy embeds the site's player, which is not reached from a test.
    await page.route(/youtube-nocookie\.com/, (route) => route.abort())
  })

  test.afterEach(() => {
    deleteVaultFiles(...made.splice(0))
  })

  test('Delete downloaded video takes the copy off the disk, and Download video brings it back', async ({
    page,
  }) => {
    made.push(getLinkNote('tidalmill03'))
    await openWorkspace(page)
    await importAddress(page, 'https://www.youtube.com/watch?v=tidalmill03')
    const tab = page.getByRole('tabpanel').filter({ hasText: CAPTION })
    await expect(tab.getByRole('button', { name: /^Play/ })).toBeVisible({ timeout: 30_000 })
    const copies = listCopies()

    await tab.getByRole('button', { name: 'Delete downloaded video' }).click()

    await expect(tab.getByRole('button', { name: /^Play/ })).toHaveCount(0)
    await expect(tab.getByRole('button', { name: 'Download video' })).toBeVisible()
    await expect.poll(() => listCopies()).toHaveLength(copies.length - 1)
    await expect(tab.getByText(CAPTION)).toBeVisible()

    await tab.getByRole('button', { name: 'Download video' }).click()

    await expect(tab.getByRole('button', { name: /^Play/ })).toBeVisible({ timeout: 30_000 })
    await expect.poll(() => listCopies()).toHaveLength(copies.length)
  })

  test('Download text again brings back the captions after Delete text', async ({
    page,
    request,
  }) => {
    made.push(getLinkNote('tidalmill04'))
    await openWorkspace(page)
    await importAddress(page, 'https://www.youtube.com/watch?v=tidalmill04')
    const tab = page.getByRole('tabpanel').filter({ hasText: 'tidalmill04' })
    await expect(tab.getByText(CAPTION)).toBeVisible({ timeout: 30_000 })
    await expect(tab.getByRole('button', { name: /^Play/ })).toBeVisible({ timeout: 30_000 })

    await runCommand(page, 'Delete text', /^Delete text$/)
    await expect(tab.getByText('The text of this link was deleted.')).toBeVisible()
    await expect(tab.getByText(CAPTION)).toHaveCount(0)

    await tab.getByRole('button', { name: 'Download text again' }).click()

    await expect(tab.getByText(CAPTION)).toBeVisible({ timeout: 30_000 })
    await expect(tab.getByText('The text of this link was deleted.')).toHaveCount(0)
    expect(await listFilePaths(request)).toContain(getLinkNote('tidalmill04'))
  })
})

test.describe('a link to an address that answers 404', () => {
  const made: string[] = []

  test.afterEach(() => {
    deleteVaultFiles(...made.splice(0))
  })

  test('reports that its text was not fetched', async ({ page, request }) => {
    const before = await listFilePaths(request)
    await openWorkspace(page)

    await importAddress(page, MISSING_PAGE_ADDRESS)

    const notice = page.locator('.notice[data-severity="error"]')
    await expect(notice).toContainText('its text was not fetched')
    const added = (await listFilePaths(request)).filter((one) => !before.includes(one))
    made.push(...added)
    expect(added).toEqual([expect.stringMatching(/unreachable\.numen\.test.*tide-tables\.md$/)])
  })
})
