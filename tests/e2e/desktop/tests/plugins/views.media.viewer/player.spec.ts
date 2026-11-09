import { test, expect, type Page } from '../../support/test.js'
import {
  getPaletteField,
  openWorkspace,
  runCommand,
  runCommandOverItem,
} from '../../support/workspace.js'
import { deleteVaultFiles, listPluginFiles, restartPlugin } from '../../support/media.js'
import { listFilePaths } from '../../support/vault.js'

/** The plugin that imports links, and keeps their copies in its folder of the vault. */
const IMPORTER = 'md.numen.tools.media.importer'

/** A caption the mock downloader of `-debug-mock-ml` publishes for every video. */
const CAPTION = 'The mill pond fills behind the gate.'

/** An address the proxy of `scripts/start-server.mjs` answers 404 for. */
const MISSING_PAGE_ADDRESS = 'http://unreachable.numen.test/tide-tables'

const listCopies = (): string[] => listPluginFiles(IMPORTER).filter((name) => name.endsWith('.wav'))

const getLinkNote = (id: string): string => `https---www.youtube.com-watch-v=${id}.md`

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

  test('Delete media takes the copy off the disk, and Download media brings it back', async ({
    page,
  }) => {
    made.push(getLinkNote('tidalmill03'))
    await openWorkspace(page)
    await importAddress(page, 'https://www.youtube.com/watch?v=tidalmill03', /tidalmill03/)
    await downloadText(page)
    await downloadVideo(page)
    const tab = page.getByRole('tabpanel', { name: /tidalmill03/ })
    await expect(tab.getByRole('button', { name: /^Play/ })).toBeVisible({ timeout: 30_000 })
    const copies = listCopies()

    await runCommand(page, 'Delete media', /^Delete media$/)

    await expect(tab.getByRole('button', { name: /^Play/ })).toHaveCount(0)
    await expect.poll(() => listCopies()).toHaveLength(copies.length - 1)
    await expect(tab.getByText(CAPTION)).toBeVisible()

    await runCommandOverItem(page, 'Download media', /^Download media$/, /^Video$/)

    await expect(tab.getByRole('button', { name: /^Play/ })).toBeVisible({ timeout: 30_000 })
    await expect.poll(() => listCopies()).toHaveLength(copies.length)
  })

  test('Download captions brings back the captions after Delete text', async ({
    page,
    request,
  }) => {
    made.push(getLinkNote('tidalmill04'))
    await openWorkspace(page)
    await importAddress(page, 'https://www.youtube.com/watch?v=tidalmill04', /tidalmill04/)
    await downloadText(page)
    await downloadVideo(page)
    const tab = page.getByRole('tabpanel', { name: /tidalmill04/ })
    await expect(tab.getByText(CAPTION)).toBeVisible({ timeout: 30_000 })
    await expect(tab.getByRole('button', { name: /^Play/ })).toBeVisible({ timeout: 30_000 })

    await runCommandOverItem(page, 'Delete text', /^Delete text$/, /\(current\)$/)
    await expect(tab.getByText(CAPTION)).toHaveCount(0)

    await runCommand(page, 'Download captions', /^Download captions$/)

    await expect(tab.getByText(CAPTION)).toBeVisible({ timeout: 30_000 })
    expect(await listFilePaths(request)).toContain(getLinkNote('tidalmill04'))
  })
})

test.describe('a link to an address that answers 404', () => {
  const made: string[] = []

  // The failed fetch stays in the list of tasks until the importer stops.
  test.afterEach(async ({ request }) => {
    deleteVaultFiles(...made.splice(0))
    await restartPlugin(request, IMPORTER)
  })

  test('reports that its text was not fetched', async ({ page, request }) => {
    const before = await listFilePaths(request)
    await openWorkspace(page)

    await importAddress(page, MISSING_PAGE_ADDRESS, /tide-tables/)
    const added = (await listFilePaths(request)).filter((one) => !before.includes(one))
    made.push(...added)
    expect(added).toEqual([expect.stringMatching(/unreachable\.numen\.test.*tide-tables\.md$/)])
    await downloadText(page)

    await page.locator('[data-tasks-status] button').click()
    const popover = page.locator('.task-popover')
    const failures = popover
      .locator('.task-item')
      .filter({ hasText: 'tide-tables answered 404 Not Found' })
    await expect(failures).toHaveCount(1)
    await expect(failures).toContainText('Fetching the text of a link')
    await expect(popover).not.toContainText('Numen is not responding')
  })
})
